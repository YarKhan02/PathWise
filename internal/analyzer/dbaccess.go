package analyzer

import (
	"go/constant"
	"go/token"
	"sort"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"

	"github.com/YarKhan02/pathwise/internal/dataflow"
	"github.com/YarKhan02/pathwise/internal/model"
)

// DBAccess is the database access reachable from a set of root functions: which
// tables are touched with which operations, and which query sites could not be
// resolved. It is the raw material for the endpoint → tables mapping.
type DBAccess struct {
	// Tables maps a table name to the sorted, de-duplicated set of operations
	// seen against it (SELECT, INSERT, UPDATE, DELETE).
	Tables map[string][]string

	// Queries are the parsed queries reached, each with the Go values bound to
	// its placeholders — the raw material the authorization findings reason over.
	Queries []ReachedQuery

	// Unresolved are query sites whose SQL could not be read or parsed — a query
	// built at runtime, most often. They are reported, never dropped: an unseen
	// query is not an absence of database access.
	Unresolved []Unresolved
}

// ReachedQuery is one database query reachable from the roots, parsed, with the
// SSA values bound to its $N placeholders (Params[0] is $1).
type ReachedQuery struct {
	SQL    string
	Query  *dataflow.Query
	Params []ssa.Value
	Func   *ssa.Function
	Pos    token.Position
}

// Unresolved is a reachable site analysis could not conclude about.
type Unresolved struct {
	Reason model.UnresolvedReason
	Detail string
	Pos    token.Position
}

// AnalyzeReachableDB walks the call graph from roots and summarizes the
// database access it reaches. roots are typically a route's handler plus its
// middleware functions, so the result is everything one endpoint touches.
//
// embeds resolves query text held in //go:embed string variables; pass a nil
// EmbedIndex when the target inlines its SQL.
func AnalyzeReachableDB(prog *ssa.Program, cg *callgraph.Graph, roots []*ssa.Function, embeds EmbedIndex) DBAccess {
	acc := DBAccess{Tables: map[string][]string{}}

	for fn := range reachableFuncs(cg, roots) {
		for _, b := range fn.Blocks {
			for _, instr := range b.Instrs {
				call, ok := instr.(*ssa.Call)
				if !ok {
					continue
				}
				queryArg, varargs, isDB := dbQueryArg(call)
				if !isDB {
					continue
				}

				pos := prog.Fset.Position(call.Pos())
				sqlText, ok := queryText(queryArg, embeds)
				if !ok {
					// A query assembled at runtime. Its text — and even which
					// tables it touches — is not recoverable here, so it is
					// recorded as dynamic_sql rather than guessed at.
					acc.Unresolved = append(acc.Unresolved, Unresolved{
						Reason: model.ReasonDynamicSQL,
						Detail: "query text is not a constant at the call site",
						Pos:    pos,
					})
					continue
				}

				parsed, err := dataflow.Parse(sqlText)
				if err != nil {
					acc.Unresolved = append(acc.Unresolved, Unresolved{
						Reason: model.ReasonDynamicSQL,
						Detail: "query text did not parse as SQL",
						Pos:    pos,
					})
					continue
				}
				for _, t := range parsed.Tables {
					acc.addOp(t, parsed.Operation)
				}
				acc.Queries = append(acc.Queries, ReachedQuery{
					SQL:    sqlText,
					Query:  parsed,
					Params: variadicValues(varargs),
					Func:   fn,
					Pos:    pos,
				})
			}
		}
	}

	return acc
}

func (a *DBAccess) addOp(table, op string) {
	if op == "" {
		return
	}
	for _, existing := range a.Tables[table] {
		if existing == op {
			return
		}
	}
	a.Tables[table] = append(a.Tables[table], op)
	sort.Strings(a.Tables[table])
}

// reachableFuncs returns every function reachable from roots over call-graph
// edges, roots included. Dependency functions appear as leaves (no out-edges),
// so the walk naturally stops at the library boundary.
func reachableFuncs(cg *callgraph.Graph, roots []*ssa.Function) map[*ssa.Function]bool {
	seen := map[*ssa.Function]bool{}
	var stack []*ssa.Function
	for _, r := range roots {
		if r != nil && !seen[r] {
			seen[r] = true
			stack = append(stack, r)
		}
	}
	for len(stack) > 0 {
		fn := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		node := cg.Nodes[fn]
		if node == nil {
			continue
		}
		for _, e := range node.Out {
			callee := e.Callee.Func
			if callee != nil && !seen[callee] {
				seen[callee] = true
				stack = append(stack, callee)
			}
		}
	}
	return seen
}

// dbQueryArg reports whether call is a database/sql query or exec that carries
// SQL text. It returns the SQL argument and the value holding the variadic bind
// arguments (the `args ...any` slice), from which Params are recovered.
// Statement methods (*sql.Stmt) are excluded: they carry no query string, so
// their first argument is a bind parameter, not SQL.
func dbQueryArg(call *ssa.Call) (sqlArg, varargs ssa.Value, ok bool) {
	callee := call.Call.StaticCallee()
	if callee == nil || callee.Pkg == nil || callee.Pkg.Pkg == nil {
		return nil, nil, false
	}
	if callee.Pkg.Pkg.Path() != "database/sql" {
		return nil, nil, false
	}
	recv := callee.Signature.Recv()
	if recv == nil || !isSQLQuerier(recv.Type()) {
		return nil, nil, false
	}

	args := call.Call.Args
	switch callee.Name() {
	case "Query", "QueryRow", "Exec":
		// (recv, query, args...) → query at index 1, variadic at 2.
		if len(args) >= 2 {
			return args[1], at(args, 2), true
		}
	case "QueryContext", "QueryRowContext", "ExecContext":
		// (recv, ctx, query, args...) → query at index 2, variadic at 3.
		if len(args) >= 3 {
			return args[2], at(args, 3), true
		}
	}
	return nil, nil, false
}

func at(args []ssa.Value, i int) ssa.Value {
	if i < len(args) {
		return args[i]
	}
	return nil
}

// isSQLQuerier reports whether t is one of the database/sql types that take a
// query string: *sql.DB, *sql.Tx, *sql.Conn. *sql.Stmt is deliberately absent.
func isSQLQuerier(t interface{ String() string }) bool {
	switch t.String() {
	case "*database/sql.DB", "*database/sql.Tx", "*database/sql.Conn":
		return true
	}
	return false
}

// queryText recovers a query's SQL at a call site: a string constant (inline
// SQL), or the contents of a //go:embed'd string variable (SQL kept in a .sql
// file). Anything else — a runtime-built string, a value from elsewhere —
// returns false and becomes a dynamic_sql site.
func queryText(v ssa.Value, embeds EmbedIndex) (string, bool) {
	if s, ok := stringConst(v); ok {
		return s, true
	}
	if g := globalOf(v); g != nil && embeds != nil {
		if text, ok := embeds[g.Object()]; ok {
			return text, true
		}
	}
	return "", false
}

// globalOf returns the package-level variable a value reads, whether the value
// is the global itself or a load of it (the usual SSA form of reading a var).
func globalOf(v ssa.Value) *ssa.Global {
	switch x := v.(type) {
	case *ssa.Global:
		return x
	case *ssa.UnOp:
		if x.Op == token.MUL {
			if g, ok := x.X.(*ssa.Global); ok {
				return g
			}
		}
	}
	return nil
}

// stringConst folds an SSA value to a string constant, including the
// concatenation of constant strings. Anything else returns false.
func stringConst(v ssa.Value) (string, bool) {
	switch x := v.(type) {
	case *ssa.Const:
		if x.Value != nil && x.Value.Kind() == constant.String {
			return constant.StringVal(x.Value), true
		}
	case *ssa.BinOp:
		if x.Op == token.ADD {
			l, lok := stringConst(x.X)
			r, rok := stringConst(x.Y)
			if lok && rok {
				return l + r, true
			}
		}
	}
	return "", false
}

// variadicValues recovers the individual values passed to a `...any` variadic
// argument — the bind parameters of a database/sql query. SSA collects them into
// an array behind a slice, and each element is boxed into an interface, so the
// elements are read from the stores into that array and unboxed. Params[0] is
// $1. Elements it cannot see through are left as the boxed value, which the
// provenance pass treats as unknown.
func variadicValues(v ssa.Value) []ssa.Value {
	if v == nil {
		return nil
	}
	slice, ok := deref(v).(*ssa.Slice)
	if !ok {
		return nil
	}
	alloc, ok := deref(slice.X).(*ssa.Alloc)
	if !ok {
		return nil
	}
	byIndex := map[int64]ssa.Value{}
	refs := alloc.Referrers()
	if refs == nil {
		return nil
	}
	for _, instr := range *refs {
		idx, ok := instr.(*ssa.IndexAddr)
		if !ok {
			continue
		}
		k, ok := idx.Index.(*ssa.Const)
		if !ok {
			continue
		}
		iref := idx.Referrers()
		if iref == nil {
			continue
		}
		for _, use := range *iref {
			if st, ok := use.(*ssa.Store); ok && st.Addr == idx {
				byIndex[k.Int64()] = unbox(st.Val)
			}
		}
	}
	out := make([]ssa.Value, 0, len(byIndex))
	for i := int64(0); i < int64(len(byIndex)); i++ {
		if val, ok := byIndex[i]; ok {
			out = append(out, val)
		} else {
			out = append(out, nil)
		}
	}
	return out
}

// deref peels a pointer load so a value read from a local reads through to the
// thing it holds.
func deref(v ssa.Value) ssa.Value {
	if u, ok := v.(*ssa.UnOp); ok && u.Op == token.MUL {
		return u.X
	}
	return v
}

// unbox removes an interface conversion so a bind parameter boxed into `any`
// shows the concrete value underneath.
func unbox(v ssa.Value) ssa.Value {
	if mi, ok := v.(*ssa.MakeInterface); ok {
		return mi.X
	}
	return v
}
