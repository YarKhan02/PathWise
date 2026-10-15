package analyzer_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/YarKhan02/pathwise/internal/analyzer"
	"github.com/YarKhan02/pathwise/internal/loader"
	"golang.org/x/tools/go/ssa"
)

// TestEmbeddedSQLResolved checks that a query whose SQL is kept in a //go:embed
// .sql file is read and attributed to the right table, rather than falling back
// to dynamic_sql — the shape AcademyEngine uses for all 300+ of its queries.
func TestEmbeddedSQLResolved(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("testdata", "embedsql"))
	if err != nil {
		t.Fatalf("resolve fixture: %v", err)
	}
	loaded, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}

	cg := analyzer.BuildCallGraph(loaded.Prog)
	funcs := analyzer.FunctionsByName(loaded.Prog)
	embeds := analyzer.BuildEmbedIndex(loaded.Initial)

	root := funcs["main.loadWidget"]
	if root == nil {
		t.Fatal("main.loadWidget not found")
	}
	acc := analyzer.AnalyzeReachableDB(loaded.Prog, cg, []*ssa.Function{root}, embeds)

	if !reflect.DeepEqual(acc.Tables, map[string][]string{"widgets": {"SELECT"}}) {
		t.Errorf("tables = %v, want {widgets: [SELECT]}", acc.Tables)
	}
	if len(acc.Unresolved) != 0 {
		t.Errorf("expected no unresolved query, got %v (embed not resolved?)", acc.Unresolved)
	}
}

// TestEmbeddedSQLWithoutIndexIsDynamic confirms the fallback is honest: with no
// embed index, the same query cannot be read and is reported as dynamic_sql
// rather than silently dropped.
func TestEmbeddedSQLWithoutIndexIsDynamic(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("testdata", "embedsql"))
	if err != nil {
		t.Fatalf("resolve fixture: %v", err)
	}
	loaded, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}

	cg := analyzer.BuildCallGraph(loaded.Prog)
	funcs := analyzer.FunctionsByName(loaded.Prog)
	root := funcs["main.loadWidget"]
	acc := analyzer.AnalyzeReachableDB(loaded.Prog, cg, []*ssa.Function{root}, nil)

	if len(acc.Tables) != 0 {
		t.Errorf("without embed index, expected no tables, got %v", acc.Tables)
	}
	if len(acc.Unresolved) != 1 {
		t.Errorf("without embed index, expected 1 dynamic_sql site, got %v", acc.Unresolved)
	}
}
