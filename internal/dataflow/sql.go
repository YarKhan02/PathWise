// Package dataflow reads SQL and Go response construction into the facts the
// authorization findings need: which columns a query constrains and with what
// value, which tables it touches, and which columns a handler serializes.
//
// SQL is parsed with pg_query_go — the real PostgreSQL parser — not a keyword
// scan, because the three findings here turn on structure a scan cannot see: a
// predicate on the tenant column, a request value reaching a primary key, a
// field backed by a sensitive column. A name match is not evidence.
package dataflow

import (
	"fmt"
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
)

// Query is the parsed shape of one SQL statement, reduced to what the findings
// reason over.
type Query struct {
	Operation string // SELECT, INSERT, UPDATE, DELETE, or "" if not one of these
	Tables    []string

	// WhereColumns is every column referenced anywhere in the WHERE clause.
	WhereColumns []string

	// EqPredicates are the equality predicates `column = value` in WHERE, which
	// is where ownership and tenancy are (or are not) enforced.
	EqPredicates []EqPredicate
}

// EqPredicate is one `column = value` comparison in a WHERE clause. Param is
// the `$N` placeholder number the column is compared against, or 0 when the
// right side is not a bind parameter (a literal, another column, a function).
type EqPredicate struct {
	Column string
	Param  int
}

// HasWhereColumn reports whether col is constrained anywhere in WHERE.
func (q *Query) HasWhereColumn(col string) bool {
	for _, c := range q.WhereColumns {
		if c == col {
			return true
		}
	}
	return false
}

// Parse parses one SQL statement. A statement pg_query cannot parse (or an
// empty one) returns an error; the caller treats that as dynamic_sql, never as
// "no predicates".
func Parse(sql string) (*Query, error) {
	res, err := pg.Parse(sql)
	if err != nil {
		return nil, fmt.Errorf("parse sql: %w", err)
	}
	if len(res.Stmts) == 0 || res.Stmts[0].Stmt == nil {
		return nil, fmt.Errorf("parse sql: no statement")
	}

	q := &Query{}
	stmt := res.Stmts[0].Stmt
	switch {
	case stmt.GetSelectStmt() != nil:
		s := stmt.GetSelectStmt()
		q.Operation = "SELECT"
		q.collectTables(s.FromClause)
		q.walkWhere(s.WhereClause)
	case stmt.GetInsertStmt() != nil:
		q.Operation = "INSERT"
		if r := stmt.GetInsertStmt().Relation; r != nil {
			q.Tables = append(q.Tables, r.Relname)
		}
	case stmt.GetUpdateStmt() != nil:
		u := stmt.GetUpdateStmt()
		q.Operation = "UPDATE"
		if u.Relation != nil {
			q.Tables = append(q.Tables, u.Relation.Relname)
		}
		q.walkWhere(u.WhereClause)
	case stmt.GetDeleteStmt() != nil:
		d := stmt.GetDeleteStmt()
		q.Operation = "DELETE"
		if d.Relation != nil {
			q.Tables = append(q.Tables, d.Relation.Relname)
		}
		q.walkWhere(d.WhereClause)
	}
	return q, nil
}

// collectTables gathers the relations named in a FROM clause, descending
// through JOINs.
func (q *Query) collectTables(from []*pg.Node) {
	for _, n := range from {
		q.collectTableNode(n)
	}
}

func (q *Query) collectTableNode(n *pg.Node) {
	if n == nil {
		return
	}
	if rv := n.GetRangeVar(); rv != nil {
		q.Tables = append(q.Tables, rv.Relname)
		return
	}
	if j := n.GetJoinExpr(); j != nil {
		q.collectTableNode(j.Larg)
		q.collectTableNode(j.Rarg)
	}
}

// walkWhere descends a WHERE clause, recording every column referenced and
// every `column = value` equality predicate.
func (q *Query) walkWhere(n *pg.Node) {
	if n == nil {
		return
	}
	if b := n.GetBoolExpr(); b != nil { // AND / OR / NOT
		for _, arg := range b.Args {
			q.walkWhere(arg)
		}
		return
	}
	if e := n.GetAExpr(); e != nil {
		lcol := columnName(e.Lexpr)
		rcol := columnName(e.Rexpr)
		for _, c := range []string{lcol, rcol} {
			if c != "" {
				q.WhereColumns = append(q.WhereColumns, c)
			}
		}
		if operatorIs(e.Name, "=") && lcol != "" {
			q.EqPredicates = append(q.EqPredicates, EqPredicate{Column: lcol, Param: paramNumber(e.Rexpr)})
		}
		return
	}
	// Any other expression shape: still record a bare column reference if this
	// node is one, so tenant/ownership presence checks are not fooled by a
	// predicate written in an unusual form.
	if c := columnName(n); c != "" {
		q.WhereColumns = append(q.WhereColumns, c)
	}
}

// columnName returns the (unqualified) column a node references, or "" if it is
// not a column reference. For a qualified name like u.id it returns the last
// segment, id.
func columnName(n *pg.Node) string {
	if n == nil {
		return ""
	}
	cr := n.GetColumnRef()
	if cr == nil {
		return ""
	}
	fields := cr.Fields
	if len(fields) == 0 {
		return ""
	}
	last := fields[len(fields)-1]
	if s := last.GetString_(); s != nil {
		return strings.ToLower(s.Sval)
	}
	return ""
}

// paramNumber returns the $N placeholder number a node is, or 0 if it is not a
// bind parameter.
func paramNumber(n *pg.Node) int {
	if n == nil {
		return 0
	}
	if p := n.GetParamRef(); p != nil {
		return int(p.Number)
	}
	return 0
}

// operatorIs reports whether an A_Expr operator name is op (e.g. "=").
func operatorIs(name []*pg.Node, op string) bool {
	if len(name) == 0 {
		return false
	}
	if s := name[len(name)-1].GetString_(); s != nil {
		return s.Sval == op
	}
	return false
}
