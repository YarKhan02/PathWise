package findings_test

import (
	"path/filepath"
	"testing"

	ginadapter "github.com/YarKhan02/pathwise/internal/adapters/gin"
	"github.com/YarKhan02/pathwise/internal/analyzer"
	"github.com/YarKhan02/pathwise/internal/authorization"
	"github.com/YarKhan02/pathwise/internal/config"
	"github.com/YarKhan02/pathwise/internal/findings"
	"github.com/YarKhan02/pathwise/internal/loader"
	"golang.org/x/tools/go/ssa"
)

// TestOwnershipCheck is the IDOR slice against the corpus. The schema keys are
// supplied directly (what a live scan reads from --db), so the test needs no
// database; it exercises the provenance, ownership-guard and gating logic.
func TestOwnershipCheck(t *testing.T) {
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

	cg := analyzer.BuildCallGraph(loaded.Prog)
	funcs := analyzer.FunctionsByName(loaded.Prog)
	embeds := analyzer.BuildEmbedIndex(loaded.Initial)
	idf := map[string]bool{}
	for _, n := range cfg.IdentityFuncs {
		idf[n] = true
	}
	cls := analyzer.NewClassifier(cg, ginadapter.IsRequestSource, idf)
	keyCols := map[string]map[string]bool{
		"students": {"id": true}, "batches": {"id": true},
		"sessions": {"id": true, "token": true}, "users": {"id": true, "email": true},
	}

	idorCount := func(handler string, authenticated bool) int {
		fn := funcs[handler]
		if fn == nil {
			t.Fatalf("handler %s not found", handler)
		}
		acc := analyzer.AnalyzeReachableDB(loaded.Prog, cg, []*ssa.Function{fn}, embeds)
		hasGuard := authorization.HasOwnershipGuard(fn, ginadapter.IsResponseWrite, cls.IsIdentity)
		return len(findings.OwnershipCheck("x", "ep", authenticated, hasGuard, acc.Queries, keyCols, cls, true))
	}

	// The seeded IDOR: request id → students PK, no identity predicate, no guard.
	if n := idorCount("main.GetStudent", true); n != 1 {
		t.Errorf("GetStudent: IDOR findings = %d, want 1", n)
	}
	// Ownership enforced via CanAccessStudent → guarded, no finding.
	if n := idorCount("main.GetStudentSafe", true); n != 0 {
		t.Errorf("GetStudentSafe: want 0 (CanAccessStudent dominates), got %d", n)
	}
	// Ownership enforced inline via user.OrgID → guarded, no finding.
	if n := idorCount("main.GetStudentProfile", true); n != 0 {
		t.Errorf("GetStudentProfile: want 0 (inline org check), got %d", n)
	}
	// Public login: unauthenticated, so its unique-email lookup is not an IDOR.
	if n := idorCount("main.Login", false); n != 0 {
		t.Errorf("Login: want 0 (public), got %d", n)
	}
	// Batch list has no key predicate at all → not an IDOR (it's the tenant case).
	if n := idorCount("main.ListBatches", true); n != 0 {
		t.Errorf("ListBatches: want 0, got %d", n)
	}
}
