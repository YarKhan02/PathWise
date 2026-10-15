package analyzer_test

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	ginadapter "github.com/YarKhan02/pathwise/internal/adapters/gin"
	"github.com/YarKhan02/pathwise/internal/analyzer"
	"github.com/YarKhan02/pathwise/internal/loader"
	"github.com/YarKhan02/pathwise/internal/model"
	"golang.org/x/tools/go/ssa"
)

// wantTables is the known answer for basic-gin: for each endpoint, the tables
// reachable from the handler and its middleware, each with its operations.
// RequireAuth resolves the session, so every authenticated endpoint also
// touches sessions and users; /login has no middleware.
var wantTables = map[string]map[string][]string{
	"POST /login": {
		"users":    {"SELECT"},
		"sessions": {"INSERT"},
	},
	"GET /students/{id}": {
		"students": {"SELECT"}, "sessions": {"SELECT"}, "users": {"SELECT"},
	},
	"GET /students/{id}/safe": {
		"students": {"SELECT"}, "sessions": {"SELECT"}, "users": {"SELECT"},
	},
	"GET /students/{id}/profile": {
		"students": {"SELECT"}, "sessions": {"SELECT"}, "users": {"SELECT"},
	},
	"GET /batches": {
		"batches": {"SELECT"}, "sessions": {"SELECT"}, "users": {"SELECT"},
	},
	"GET /dashboard": {
		"batches": {"SELECT"}, "sessions": {"SELECT"}, "users": {"SELECT"},
	},
	"GET /admin/reports": {
		"batches": {"SELECT"}, "sessions": {"SELECT"}, "users": {"SELECT"},
	},
	// /search/students builds its query with fmt.Sprintf, so no table is
	// claimed; it is asserted separately as an unresolved dynamic_sql site.
	"GET /search/students": {
		"sessions": {"SELECT"}, "users": {"SELECT"},
	},
}

// TestEndpointTableAccess is the Phase 2 exit criterion: the correct table set
// per corpus endpoint, with every unresolved query site named.
func TestEndpointTableAccess(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "testdata", "apps", "basic-gin"))
	if err != nil {
		t.Fatalf("resolve corpus: %v", err)
	}
	loaded, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("load corpus: %v", err)
	}

	routes := ginadapter.Discover(loaded.Prog, loaded.Packages).Routes
	cg := analyzer.BuildCallGraph(loaded.Prog)
	funcs := analyzer.FunctionsByName(loaded.Prog)
	embeds := analyzer.BuildEmbedIndex(loaded.Initial)

	for _, r := range routes {
		roots := rootsFor(t, funcs, r)
		acc := analyzer.AnalyzeReachableDB(loaded.Prog, cg, roots, embeds)

		want, ok := wantTables[r.Subject()]
		if !ok {
			t.Errorf("route %s not in expected table map", r.Subject())
			continue
		}
		if !reflect.DeepEqual(normalize(acc.Tables), normalize(want)) {
			t.Errorf("route %s tables =\n  %v\nwant\n  %v", r.Subject(), acc.Tables, want)
		}
	}

	// The dynamic-SQL endpoint must surface a dynamic_sql miss, not silence.
	searchRoots := rootsFor(t, funcs, routeNamed(t, routes, "GET /search/students"))
	search := analyzer.AnalyzeReachableDB(loaded.Prog, cg, searchRoots, embeds)
	foundDynamic := false
	for _, u := range search.Unresolved {
		if u.Reason == model.ReasonDynamicSQL {
			foundDynamic = true
		}
	}
	if !foundDynamic {
		t.Errorf("GET /search/students: expected a dynamic_sql unresolved site, got %+v", search.Unresolved)
	}
}

func rootsFor(t *testing.T, funcs map[string]*ssa.Function, r ginadapter.Route) []*ssa.Function {
	t.Helper()
	var roots []*ssa.Function
	for _, name := range append([]string{r.Handler}, r.MiddlewareNames()...) {
		fn := funcs[name]
		if fn == nil {
			t.Fatalf("route %s: function %q not found in program", r.Subject(), name)
		}
		roots = append(roots, fn)
	}
	return roots
}

func routeNamed(t *testing.T, routes []ginadapter.Route, subject string) ginadapter.Route {
	t.Helper()
	for _, r := range routes {
		if r.Subject() == subject {
			return r
		}
	}
	t.Fatalf("route %s not discovered", subject)
	return ginadapter.Route{}
}

// normalize sorts each operation slice so comparison is order-independent.
func normalize(m map[string][]string) map[string][]string {
	out := make(map[string][]string, len(m))
	for k, v := range m {
		cp := append([]string(nil), v...)
		sort.Strings(cp)
		out[k] = cp
	}
	return out
}
