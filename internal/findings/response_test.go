package findings_test

import (
	"path/filepath"
	"testing"

	ginadapter "github.com/YarKhan02/PathWise/internal/adapters/gin"
	"github.com/YarKhan02/PathWise/internal/analyzer"
	"github.com/YarKhan02/PathWise/internal/config"
	"github.com/YarKhan02/PathWise/internal/findings"
	"github.com/YarKhan02/PathWise/internal/loader"
	"github.com/YarKhan02/PathWise/internal/model"
)

func TestResponseScope(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "testdata", "apps", "basic-gin"))
	if err != nil {
		t.Fatalf("resolve corpus: %v", err)
	}
	loaded, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("load corpus: %v", err)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	funcs := analyzer.FunctionsByName(loaded.Prog)

	// GetStudentProfile serializes national_id, which is sensitive and outside
	// this endpoint's scope → one finding.
	fs := findings.ResponseScope("x", "GET /students/{id}/profile",
		funcs["main.GetStudentProfile"], []string{"students"}, cfg, ginadapter.ResponseBody)
	if len(fs) != 1 || fs[0].Kind != model.FindingResponseFieldExceedsScope {
		t.Fatalf("GetStudentProfile: findings = %+v, want one response_field_exceeds_scope", fs)
	}
	if !contains(fs[0].StableKey, "students.national_id") {
		t.Errorf("finding should concern students.national_id, got %q", fs[0].StableKey)
	}

	// GetStudent serializes Student (email is in scope, no national_id) → none.
	if fs := findings.ResponseScope("x", "GET /students/{id}",
		funcs["main.GetStudent"], []string{"students"}, cfg, ginadapter.ResponseBody); len(fs) != 0 {
		t.Errorf("GetStudent should not exceed scope: %+v", fs)
	}

	// ListBatches serializes Batch (no sensitive columns) → none.
	if fs := findings.ResponseScope("x", "GET /batches",
		funcs["main.ListBatches"], []string{"batches"}, cfg, ginadapter.ResponseBody); len(fs) != 0 {
		t.Errorf("ListBatches should not exceed scope: %+v", fs)
	}

	// GetStudentSafe serializes email, which is sensitive — but /students/{id}/safe
	// has no declared scope, so there is no scope to exceed and nothing fires.
	if fs := findings.ResponseScope("x", "GET /students/{id}/safe",
		funcs["main.GetStudentSafe"], []string{"students"}, cfg, ginadapter.ResponseBody); len(fs) != 0 {
		t.Errorf("endpoint with no declared scope must not be flagged: %+v", fs)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
