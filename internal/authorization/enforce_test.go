package authorization_test

import (
	"path/filepath"
	"testing"

	ginadapter "github.com/YarKhan02/pathwise/internal/adapters/gin"
	"github.com/YarKhan02/pathwise/internal/analyzer"
	"github.com/YarKhan02/pathwise/internal/authorization"
	"github.com/YarKhan02/pathwise/internal/config"
	"github.com/YarKhan02/pathwise/internal/loader"
)

// TestCheckEnforcement is the Phase 3 role_check_unenforced slice against the
// corpus. The two seeded flaws must read as unenforced for the right reason,
// and the negative control must read as enforced so it produces no finding.
func TestCheckEnforcement(t *testing.T) {
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

	checkFuncs := map[string]bool{}
	for _, name := range cfg.AuthCheckFuncs {
		checkFuncs[name] = true
	}
	funcs := analyzer.FunctionsByName(loaded.Prog)

	// handler -> expected (check func -> enforcement).
	cases := map[string]map[string]authorization.Enforcement{
		"main.AdminReport":       {"main.IsAdmin": authorization.Discarded},         // _ = IsAdmin(user)
		"main.TeacherDashboard":  {"main.HasRole": authorization.NonDominating},     // guarded only when strict=1
		"main.GetStudentSafe":    {"main.CanAccessStudent": authorization.Enforced}, // abort dominates c.JSON
		"main.GetStudent":        {},                                                // idor: no check at all
		"main.GetStudentProfile": {},                                                // inline compare, not a check call
		"main.Login":             {},
		"main.ListBatches":       {},
	}

	for handler, want := range cases {
		fn := funcs[handler]
		if fn == nil {
			t.Errorf("handler %s not found", handler)
			continue
		}
		got := map[string]authorization.Enforcement{}
		for _, c := range authorization.CheckEnforcement(fn, checkFuncs, ginadapter.IsResponseWrite) {
			got[c.Func] = c.Enforcement
		}
		if len(got) != len(want) {
			t.Errorf("%s: found checks %v, want %v", handler, got, want)
			continue
		}
		for name, wantE := range want {
			if got[name] != wantE {
				t.Errorf("%s: check %s enforcement = %q, want %q", handler, name, got[name], wantE)
			}
		}
	}
}
