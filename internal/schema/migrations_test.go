package schema_test

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/YarKhan02/pathwise/internal/corpus"
	"github.com/YarKhan02/pathwise/internal/schema"
)

// TestReconstructMatchesCorpus checks the schema rebuilt from the corpus
// migrations matches expected.yaml — the same ground truth as the live reader,
// but parsed from DDL with no database.
func TestReconstructMatchesCorpus(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "testdata", "apps", "basic-gin"))
	if err != nil {
		t.Fatalf("resolve corpus: %v", err)
	}
	got, err := schema.Reconstruct(filepath.Join(dir, "migrations"))
	if err != nil {
		t.Fatalf("reconstruct: %v", err)
	}
	app, err := corpus.LoadApp(dir)
	if err != nil {
		t.Fatalf("load corpus: %v", err)
	}
	want := app.Schema

	if len(got.Tables) != len(want.Tables) {
		t.Errorf("tables: got %d, want %d", len(got.Tables), len(want.Tables))
	}
	for name, wt := range want.Tables {
		gt := got.Find(name)
		if gt == nil {
			t.Errorf("table %q not reconstructed", name)
			continue
		}
		if !sameSet(columnNames(gt), wt.Columns) {
			t.Errorf("table %q columns: got %v, want %v", name, columnNames(gt), wt.Columns)
		}
		if !sameSet(gt.PrimaryKey, wt.PrimaryKey) {
			t.Errorf("table %q PK: got %v, want %v", name, gt.PrimaryKey, wt.PrimaryKey)
		}
	}

	gotFK := map[string]bool{}
	for _, fk := range got.ForeignKeys {
		gotFK[fk.FromTable+"."+fk.FromColumn+" -> "+fk.ToTable+"."+fk.ToColumn] = true
	}
	for _, wfk := range want.ForeignKeys {
		if edge := wfk.From + " -> " + wfk.To; !gotFK[edge] {
			t.Errorf("foreign key %q not reconstructed (got %v)", edge, keysOf(gotFK))
		}
	}
	if len(got.ForeignKeys) != len(want.ForeignKeys) {
		t.Errorf("foreign keys: got %d, want %d", len(got.ForeignKeys), len(want.ForeignKeys))
	}

	// The reconstructed schema must also know the keys IDOR relies on.
	students := got.Find("students")
	if students == nil || len(students.PrimaryKey) != 1 || students.PrimaryKey[0] != "id" {
		t.Errorf("students primary key should be [id], got %+v", students)
	}
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
