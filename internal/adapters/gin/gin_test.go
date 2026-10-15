package gin_test

import (
	"path/filepath"
	"strings"
	"testing"

	ginadapter "github.com/YarKhan02/pathwise/internal/adapters/gin"
	"github.com/YarKhan02/pathwise/internal/corpus"
	"github.com/YarKhan02/pathwise/internal/loader"
	"github.com/YarKhan02/pathwise/internal/model"
)

// corpusApp is the basic-gin app, resolved relative to this test file.
func corpusApp(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", "..", "testdata", "apps", "basic-gin"))
	if err != nil {
		t.Fatalf("resolve corpus path: %v", err)
	}
	return dir
}

// TestDiscoverBasicGinRoutes is the Phase 1 exit criterion for route discovery:
// every route the corpus declares must be recovered with the right handler and
// middleware chain, and nothing must be invented. The corpus's expected.yaml is
// the ground truth, so the test reads it rather than hard-coding a second copy.
func TestDiscoverBasicGinRoutes(t *testing.T) {
	dir := corpusApp(t)

	app, err := corpus.LoadApp(dir)
	if err != nil {
		t.Fatalf("load corpus expectations: %v", err)
	}

	loaded, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("load corpus app: %v", err)
	}
	if errs := loaded.BuildErrors(); len(errs) > 0 {
		// The corpus app must compile for go/packages to analyze it. If this
		// fires, route discovery is being measured against a broken load.
		t.Fatalf("corpus app did not type-check: %v", errs)
	}

	res := ginadapter.Discover(loaded.Prog, loaded.Packages)

	got := map[string]ginadapter.Route{}
	for _, r := range res.Routes {
		if _, dup := got[r.Subject()]; dup {
			t.Errorf("route %s discovered more than once", r.Subject())
		}
		got[r.Subject()] = r
	}

	// Every expected route is present, with matching handler and middleware.
	for _, want := range app.Routes {
		r, ok := got[want.Subject()]
		if !ok {
			t.Errorf("route %s not discovered", want.Subject())
			continue
		}
		if r.Handler != want.Handler {
			t.Errorf("route %s: handler = %q, want %q", want.Subject(), r.Handler, want.Handler)
		}
		if !equalStrings(r.MiddlewareNames(), want.Middleware) {
			t.Errorf("route %s: middleware = %v, want %v", want.Subject(), r.MiddlewareNames(), want.Middleware)
		}
	}

	// Nothing invented: no discovered route lacks an expected counterpart.
	expected := map[string]bool{}
	for _, want := range app.Routes {
		expected[want.Subject()] = true
	}
	for subject := range got {
		if !expected[subject] {
			t.Errorf("discovered route %s is not declared in expected.yaml (false positive)", subject)
		}
	}

	// Exit criterion: at least 95% of declared routes recovered.
	recovered := 0
	for _, want := range app.Routes {
		if _, ok := got[want.Subject()]; ok {
			recovered++
		}
	}
	if ratio := float64(recovered) / float64(len(app.Routes)); ratio < 0.95 {
		t.Errorf("recovered %d/%d routes (%.0f%%), want >= 95%%",
			recovered, len(app.Routes), ratio*100)
	}
}

// TestMiddlewareOrderPreserved checks that the admin route reports its two
// middlewares in execution order, since ordering is what a dominance proof in
// Phase 3 will depend on.
func TestMiddlewareOrderPreserved(t *testing.T) {
	dir := corpusApp(t)
	loaded, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("load corpus app: %v", err)
	}
	res := ginadapter.Discover(loaded.Prog, loaded.Packages)

	want := model.RouteSubject("GET", "/admin/reports")
	for _, r := range res.Routes {
		if r.Subject() == want {
			if !equalStrings(r.MiddlewareNames(), []string{"main.RequireAuth", "main.RequireRole"}) {
				t.Fatalf("%s middleware = %v, want [main.RequireAuth main.RequireRole]", want, r.MiddlewareNames())
			}
			// The role argument of RequireRole("admin") must be captured.
			for _, m := range r.Middleware {
				if m.Name == "main.RequireRole" && !equalStrings(m.Args, []string{"admin"}) {
					t.Fatalf("%s RequireRole args = %v, want [admin]", want, m.Args)
				}
			}
			return
		}
	}
	t.Fatalf("route %s not found", want)
}

// TestUnresolvableHandlerIsMiss checks that a route whose handler cannot be
// identified is reported as an unresolved miss, not emitted with an empty
// handler. testdata/ginspread registers "/spread" with handlers spread from a
// function result (opaque to value tracking) and "/plain" with a direct
// handler; the first must be a miss and the second a clean route.
func TestUnresolvableHandlerIsMiss(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("testdata", "ginspread"))
	if err != nil {
		t.Fatalf("resolve fixture: %v", err)
	}
	loaded, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	res := ginadapter.Discover(loaded.Prog, loaded.Packages)

	emitted := map[string]string{} // subject -> handler
	for _, r := range res.Routes {
		emitted[r.Subject()] = r.Handler
	}

	plain := model.RouteSubject("GET", "/plain")
	if emitted[plain] != "main.show" {
		t.Errorf("%s: handler = %q, want main.show", plain, emitted[plain])
	}

	spread := model.RouteSubject("GET", "/spread")
	if _, ok := emitted[spread]; ok {
		t.Errorf("%s was emitted as a route; an unresolvable handler must be a miss, not a handler-less route", spread)
	}

	foundMiss := false
	for _, m := range res.Misses {
		if m.Reason == model.ReasonDynamicRoute && strings.Contains(m.Detail, "/spread") {
			foundMiss = true
			break
		}
	}
	if !foundMiss {
		t.Errorf("expected a dynamic_route miss for /spread, got misses %+v", res.Misses)
	}
}

// TestRouterParameterModulePattern checks the real-world shape AcademyEngine
// uses: routes registered inside a method that receives the engine as a
// parameter, with middleware on the group and method-bound handlers. The engine
// is built in a different function, so the router is only a parameter here.
func TestRouterParameterModulePattern(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("testdata", "ginmodule"))
	if err != nil {
		t.Fatalf("resolve fixture: %v", err)
	}
	loaded, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	res := ginadapter.Discover(loaded.Prog, loaded.Packages)

	got := map[string]ginadapter.Route{}
	for _, r := range res.Routes {
		got[r.Subject()] = r
	}

	want := []struct {
		subject, handler string
	}{
		{model.RouteSubject("GET", "/course"), "main.GetCourse"},
		{model.RouteSubject("POST", "/course"), "main.InsertCourse"},
	}
	for _, w := range want {
		r, ok := got[w.subject]
		if !ok {
			t.Errorf("route %s not discovered (router-as-parameter still unresolved?)", w.subject)
			continue
		}
		if r.Handler != w.handler {
			t.Errorf("route %s: handler = %q, want %q", w.subject, r.Handler, w.handler)
		}
		if !equalStrings(r.MiddlewareNames(), []string{"main.Auth", "main.RequireRole"}) {
			t.Errorf("route %s: middleware = %v, want [main.Auth main.RequireRole]", w.subject, r.MiddlewareNames())
		}
		for _, m := range r.Middleware {
			if m.Name == "main.RequireRole" && !equalStrings(m.Args, []string{"admin"}) {
				t.Errorf("route %s: RequireRole args = %v, want [admin]", w.subject, m.Args)
			}
		}
	}
}

// equalStrings compares two slices positionally. Middleware order is part of
// the chain's meaning, so this is a strict compare, not a set compare. A nil
// slice and an empty slice are treated as equal (both mean "no middleware").
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
