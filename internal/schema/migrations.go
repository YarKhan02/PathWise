package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
)

// Reconstruct rebuilds the schema by parsing a project's migration SQL in file
// order. It is the degraded source used when no live database is available: a
// replayed schema can be wrong where migrations were edited or not all run, so
// facts from it are marked inferred, and the live catalog is always preferred.
//
// It applies CREATE TABLE in full (columns, primary/unique keys, inline and
// table-level foreign keys) and the common ALTER TABLE ADD COLUMN / ADD
// CONSTRAINT. A statement it does not model is skipped rather than guessed at.
func Reconstruct(dir string) (*Schema, error) {
	entries, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("read migrations %s: %w", dir, err)
	}
	sort.Strings(entries) // golang-migrate applies in filename order

	s := &Schema{}
	byName := map[string]*Table{}
	for _, f := range entries {
		if strings.HasSuffix(f, ".down.sql") {
			continue // down migrations undo; replaying them would corrupt the schema
		}
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f, err)
		}
		res, err := pg.Parse(string(data))
		if err != nil {
			continue // a migration we cannot parse is skipped, not fatal
		}
		for _, raw := range res.Stmts {
			applyStmt(raw.Stmt, s, byName)
		}
	}
	return s, nil
}

func applyStmt(stmt *pg.Node, s *Schema, byName map[string]*Table) {
	switch {
	case stmt.GetCreateStmt() != nil:
		applyCreate(stmt.GetCreateStmt(), s, byName)
	case stmt.GetAlterTableStmt() != nil:
		applyAlter(stmt.GetAlterTableStmt(), s, byName)
	}
}

func applyCreate(c *pg.CreateStmt, s *Schema, byName map[string]*Table) {
	if c.Relation == nil {
		return
	}
	name := c.Relation.Relname
	t := &Table{Name: name}
	for _, elt := range c.TableElts {
		if col := elt.GetColumnDef(); col != nil {
			addColumn(t, s, name, col)
		} else if con := elt.GetConstraint(); con != nil {
			addTableConstraint(t, s, name, con)
		}
	}
	s.Tables = append(s.Tables, *t)
	byName[name] = &s.Tables[len(s.Tables)-1]
}

func addColumn(t *Table, s *Schema, table string, col *pg.ColumnDef) {
	c := Column{Name: col.Colname, Type: typeName(col.TypeName), Nullable: true}
	for _, cn := range col.Constraints {
		con := cn.GetConstraint()
		if con == nil {
			continue
		}
		switch con.Contype {
		case pg.ConstrType_CONSTR_NOTNULL:
			c.Nullable = false
		case pg.ConstrType_CONSTR_PRIMARY:
			c.Nullable = false
			t.PrimaryKey = append(t.PrimaryKey, col.Colname)
		case pg.ConstrType_CONSTR_UNIQUE:
			t.Unique = append(t.Unique, []string{col.Colname})
		case pg.ConstrType_CONSTR_FOREIGN:
			if con.Pktable != nil {
				s.ForeignKeys = append(s.ForeignKeys, ForeignKey{
					FromTable: table, FromColumn: col.Colname,
					ToTable: con.Pktable.Relname, ToColumn: firstName(con.PkAttrs),
				})
			}
		}
	}
	t.Columns = append(t.Columns, c)
}

// addTableConstraint handles table-level PRIMARY KEY / UNIQUE / FOREIGN KEY.
func addTableConstraint(t *Table, s *Schema, table string, con *pg.Constraint) {
	switch con.Contype {
	case pg.ConstrType_CONSTR_PRIMARY:
		t.PrimaryKey = append(t.PrimaryKey, names(con.Keys)...)
	case pg.ConstrType_CONSTR_UNIQUE:
		if cols := names(con.Keys); len(cols) > 0 {
			t.Unique = append(t.Unique, cols)
		}
	case pg.ConstrType_CONSTR_FOREIGN:
		from := names(con.FkAttrs)
		to := names(con.PkAttrs)
		if con.Pktable != nil {
			for i := range from {
				toCol := ""
				if i < len(to) {
					toCol = to[i]
				}
				s.ForeignKeys = append(s.ForeignKeys, ForeignKey{
					FromTable: table, FromColumn: from[i], ToTable: con.Pktable.Relname, ToColumn: toCol,
				})
			}
		}
	}
}

// applyAlter handles ALTER TABLE ADD COLUMN and ADD CONSTRAINT, enough to carry
// forward a schema split across migrations. Other alterations are skipped.
func applyAlter(a *pg.AlterTableStmt, s *Schema, byName map[string]*Table) {
	if a.Relation == nil {
		return
	}
	t := byName[a.Relation.Relname]
	if t == nil {
		return
	}
	for _, cmdNode := range a.Cmds {
		cmd := cmdNode.GetAlterTableCmd()
		if cmd == nil || cmd.Def == nil {
			continue
		}
		if col := cmd.Def.GetColumnDef(); col != nil {
			addColumn(t, s, t.Name, col)
		} else if con := cmd.Def.GetConstraint(); con != nil {
			addTableConstraint(t, s, t.Name, con)
		}
	}
}

func typeName(tn *pg.TypeName) string {
	if tn == nil {
		return ""
	}
	return firstName(reverse(tn.Names)) // last segment is the type (pg_catalog.int8 → int8)
}

func reverse(ns []*pg.Node) []*pg.Node {
	out := make([]*pg.Node, len(ns))
	for i := range ns {
		out[i] = ns[len(ns)-1-i]
	}
	return out
}

func names(ns []*pg.Node) []string {
	var out []string
	for _, n := range ns {
		if s := n.GetString_(); s != nil {
			out = append(out, s.Sval)
		}
	}
	return out
}

func firstName(ns []*pg.Node) string {
	for _, n := range ns {
		if s := n.GetString_(); s != nil {
			return s.Sval
		}
	}
	return ""
}
