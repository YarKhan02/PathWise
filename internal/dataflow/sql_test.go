package dataflow

import (
	"reflect"
	"testing"
)

func TestParseCorpusQueries(t *testing.T) {
	cases := []struct {
		sql      string
		op       string
		tables   []string
		eq       []EqPredicate
		whereCol []string // a subset that must be present
	}{
		{
			sql:    "SELECT id, org_id, batch_id, name, email FROM students WHERE id = $1",
			op:     "SELECT",
			tables: []string{"students"},
			eq:     []EqPredicate{{Column: "id", Param: 1}},
		},
		{
			sql:    "SELECT id, org_id, name FROM batches ORDER BY name",
			op:     "SELECT",
			tables: []string{"batches"},
			eq:     nil,
		},
		{
			sql:    "SELECT id, org_id, name FROM batches WHERE org_id = $1",
			op:     "SELECT",
			tables: []string{"batches"},
			eq:     []EqPredicate{{Column: "org_id", Param: 1}},
		},
		{
			sql:      "SELECT u.id, u.org_id, u.role FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.token = $1 AND s.expires_at > now()",
			op:       "SELECT",
			tables:   []string{"sessions", "users"},
			whereCol: []string{"token", "expires_at"},
		},
		{
			sql:    "INSERT INTO sessions (user_id, token, expires_at) VALUES ($1, $2, now()) RETURNING token",
			op:     "INSERT",
			tables: []string{"sessions"},
		},
	}

	for _, c := range cases {
		q, err := Parse(c.sql)
		if err != nil {
			t.Errorf("parse %q: %v", c.sql, err)
			continue
		}
		if q.Operation != c.op {
			t.Errorf("%q: op = %q, want %q", c.sql, q.Operation, c.op)
		}
		if !reflect.DeepEqual(q.Tables, c.tables) {
			t.Errorf("%q: tables = %v, want %v", c.sql, q.Tables, c.tables)
		}
		if c.eq != nil && !reflect.DeepEqual(q.EqPredicates, c.eq) {
			t.Errorf("%q: eq predicates = %+v, want %+v", c.sql, q.EqPredicates, c.eq)
		}
		for _, col := range c.whereCol {
			if !q.HasWhereColumn(col) {
				t.Errorf("%q: WHERE should reference %q, got %v", c.sql, col, q.WhereColumns)
			}
		}
	}
}

func TestReadWriteColumns(t *testing.T) {
	colNames := func(refs []ColumnRef) []string {
		out := make([]string, len(refs))
		for i, r := range refs {
			out[i] = r.Name
		}
		return out
	}

	// SELECT projects read columns.
	q, _ := Parse("SELECT id, name, email FROM students WHERE id = $1")
	if got := colNames(q.ReadColumns); !reflect.DeepEqual(got, []string{"id", "name", "email"}) {
		t.Errorf("read columns = %v, want [id name email]", got)
	}
	if q.Star {
		t.Error("SELECT of named columns must not set Star")
	}

	// SELECT * is honest: Star set, no named read columns.
	q, _ = Parse("SELECT * FROM students WHERE id = $1")
	if !q.Star || len(q.ReadColumns) != 0 {
		t.Errorf("SELECT *: star=%v read=%v, want star=true read=[]", q.Star, q.ReadColumns)
	}

	// INSERT records written columns, not a star.
	q, _ = Parse("INSERT INTO sessions (user_id, token, expires_at) VALUES ($1,$2,$3)")
	if got := colNames(q.WriteColumns); !reflect.DeepEqual(got, []string{"user_id", "token", "expires_at"}) {
		t.Errorf("insert write columns = %v, want [user_id token expires_at]", got)
	}

	// UPDATE records SET targets as writes and keeps the WHERE column as filter.
	q, _ = Parse("UPDATE students SET name = $1 WHERE id = $2")
	if got := colNames(q.WriteColumns); !reflect.DeepEqual(got, []string{"name"}) {
		t.Errorf("update write columns = %v, want [name]", got)
	}
	if !q.HasWhereColumn("id") {
		t.Errorf("update WHERE should reference id, got %v", q.WhereColumns)
	}
}

func TestDynamicSQLUnparsable(t *testing.T) {
	// A fragment with an unquoted interpolation hole does not parse; the caller
	// must treat that as dynamic_sql, so Parse must return an error.
	if _, err := Parse("SELECT id, name FROM students WHERE name LIKE '%"); err == nil {
		t.Error("expected parse error for a malformed/dynamic query")
	}
}
