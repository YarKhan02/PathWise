package analyzer

import (
	"go/token"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
)

// Provenance classifies where a value came from — the pivot of IDOR detection.
// A request-derived value reaching a primary-key predicate with no
// identity-derived predicate alongside it is the signature of an ownership
// failure. Unknown is neither safe nor unsafe: it produces no finding.
type Provenance string

const (
	ProvRequest  Provenance = "request"
	ProvIdentity Provenance = "identity"
	ProvConstant Provenance = "constant"
	ProvUnknown  Provenance = "unknown"
)

// Classifier resolves provenance over SSA, following a helper's parameters back
// to the arguments its callers pass (the value bound to WHERE id = $1 is usually
// a parameter of the query helper, request-derived one call up).
type Classifier struct {
	cg              *callgraph.Graph
	isRequestSource func(*ssa.Function) bool
	identityFuncs   map[string]bool
}

// NewClassifier builds a provenance classifier. isRequestSource recognizes the
// framework's request accessors; identityFuncs are the configured functions that
// return the authenticated principal (config.identity_funcs).
func NewClassifier(cg *callgraph.Graph, isRequestSource func(*ssa.Function) bool, identityFuncs map[string]bool) *Classifier {
	return &Classifier{cg: cg, isRequestSource: isRequestSource, identityFuncs: identityFuncs}
}

// Provenance classifies a value.
func (c *Classifier) Provenance(v ssa.Value) Provenance {
	return c.prov(v, map[ssa.Value]bool{}, 0)
}

// IsIdentity reports whether a value is derived from the authenticated principal.
func (c *Classifier) IsIdentity(v ssa.Value) bool { return c.Provenance(v) == ProvIdentity }

func (c *Classifier) prov(v ssa.Value, seen map[ssa.Value]bool, depth int) Provenance {
	if v == nil || depth > 14 {
		return ProvUnknown
	}
	v = peel(v)
	if seen[v] {
		return ProvUnknown
	}
	seen[v] = true

	switch x := v.(type) {
	case *ssa.Const:
		return ProvConstant
	case *ssa.Call:
		callee := x.Call.StaticCallee()
		if callee != nil {
			if c.isRequestSource(callee) {
				return ProvRequest
			}
			if c.identityFuncs[qualifiedName(callee)] {
				return ProvIdentity
			}
		}
		return ProvUnknown
	case *ssa.Parameter:
		return c.fromCallers(x, seen, depth)
	case *ssa.FieldAddr:
		return identityOrBase(c.prov(x.X, seen, depth+1))
	case *ssa.Field:
		return identityOrBase(c.prov(x.X, seen, depth+1))
	case *ssa.IndexAddr:
		return c.prov(x.X, seen, depth+1)
	case *ssa.UnOp:
		if x.Op == token.MUL { // pointer load
			return c.prov(x.X, seen, depth+1)
		}
		return ProvUnknown
	case *ssa.BinOp:
		return combine(c.prov(x.X, seen, depth+1), c.prov(x.Y, seen, depth+1))
	case *ssa.Phi:
		out := ProvUnknown
		for i, e := range x.Edges {
			p := c.prov(e, seen, depth+1)
			if i == 0 {
				out = p
			} else {
				out = combine(out, p)
			}
		}
		return out
	}
	return ProvUnknown
}

// fromCallers traces a parameter to the arguments its callers pass. If any
// caller supplies a request-derived value, the parameter is request-tainted;
// otherwise identity if a caller supplies identity; else unknown.
func (c *Classifier) fromCallers(p *ssa.Parameter, seen map[ssa.Value]bool, depth int) Provenance {
	fn := p.Parent()
	idx := -1
	for i, q := range fn.Params {
		if q == p {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ProvUnknown
	}
	node := c.cg.Nodes[fn]
	if node == nil {
		return ProvUnknown
	}
	result := ProvUnknown
	any := false
	for _, e := range node.In {
		if e.Site == nil {
			continue
		}
		args := e.Site.Common().Args
		if idx >= len(args) {
			continue
		}
		p := c.prov(args[idx], seen, depth+1)
		if p == ProvRequest {
			return ProvRequest
		}
		if !any {
			result, any = p, true
		} else {
			result = combine(result, p)
		}
	}
	return result
}

// qualifiedName renders a function as package.Function, matching the names
// config and the Gin adapter use (e.g. "main.CurrentUser").
func qualifiedName(fn *ssa.Function) string {
	if fn.Pkg != nil && fn.Pkg.Pkg != nil {
		return fn.Pkg.Pkg.Name() + "." + fn.Name()
	}
	return fn.Name()
}

// peel removes conversions that do not change where a value came from.
func peel(v ssa.Value) ssa.Value {
	for {
		switch x := v.(type) {
		case *ssa.ChangeType:
			v = x.X
		case *ssa.Convert:
			v = x.X
		case *ssa.MakeInterface:
			v = x.X
		default:
			return v
		}
	}
}

// identityOrBase keeps identity provenance through a field access (user.OrgID is
// identity-derived) and otherwise passes the base provenance through.
func identityOrBase(base Provenance) Provenance {
	return base
}

// combine merges two provenances: request dominates (taint), then identity,
// then constant, else unknown.
func combine(a, b Provenance) Provenance {
	if a == ProvRequest || b == ProvRequest {
		return ProvRequest
	}
	if a == ProvIdentity || b == ProvIdentity {
		return ProvIdentity
	}
	if a == ProvConstant && b == ProvConstant {
		return ProvConstant
	}
	return ProvUnknown
}
