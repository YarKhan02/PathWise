package report_test

import (
	"strings"
	"testing"

	"github.com/YarKhan02/PathWise/internal/model"
	"github.com/YarKhan02/PathWise/internal/report"
)

// TestBuildAndRender checks the projection joins roles, resources and
// unresolved queries onto the right endpoint, and that the HTML renders the
// role and table so the page actually shows the map.
func TestBuildAndRender(t *testing.T) {
	routes := []model.Fact{{
		Kind: model.FactRoute, Subject: "GET /course/{id}",
		Attrs: map[string]any{"method": "GET", "pattern": "/course/{id}", "handler": "course.Get"},
	}}
	middleware := []model.Fact{{
		Kind: model.FactMiddleware, Subject: "course.RequireRole",
		Attrs: map[string]any{"route": "GET /course/{id}", "args": []string{"admin"}},
	}}
	queries := []model.Fact{
		{
			Kind: model.FactQuery, Subject: "GET /course/{id} -> course",
			Attrs:    map[string]any{"endpoint": "GET /course/{id}", "table": "course", "operations": []string{"SELECT"}},
			Evidence: model.Inferred("reachable table"),
		},
		{
			Kind: model.FactQuery, Subject: "GET /course/{id} -> (unresolved query)",
			Attrs:    map[string]any{"endpoint": "GET /course/{id}"},
			Evidence: model.Unresolved(model.ReasonDynamicSQL, "runtime-built"),
		},
	}

	rep := report.Build("demo", "scan1", "none", "now", routes, middleware, queries, nil, nil, nil, nil, nil, nil)

	if rep.RouteCount != 1 || rep.RoleCount != 1 || rep.TableCount != 1 || rep.UnresolvedCount != 1 {
		t.Fatalf("counts: routes=%d roles=%d tables=%d unresolved=%d, want 1/1/1/1",
			rep.RouteCount, rep.RoleCount, rep.TableCount, rep.UnresolvedCount)
	}
	ep := rep.Endpoints[0]
	if len(ep.Roles) != 1 || ep.Roles[0] != "admin" {
		t.Errorf("roles = %v, want [admin]", ep.Roles)
	}
	if len(ep.Resources) != 1 || ep.Resources[0].Table != "course" {
		t.Errorf("resources = %+v, want course", ep.Resources)
	}
	if !ep.Unresolved {
		t.Error("endpoint should be flagged unresolved (dynamic_sql reachable)")
	}

	var sb strings.Builder
	if err := report.RenderHTML(&sb, rep); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := sb.String()
	for _, want := range []string{"admin", "course", "/course/{id}", "data-role=\"admin\""} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered HTML missing %q", want)
		}
	}
}
