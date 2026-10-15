package findings_test

import (
	"testing"

	"github.com/YarKhan02/pathwise/internal/analyzer"
	"github.com/YarKhan02/pathwise/internal/config"
	"github.com/YarKhan02/pathwise/internal/dataflow"
	"github.com/YarKhan02/pathwise/internal/findings"
	"github.com/YarKhan02/pathwise/internal/model"
)

// corpusTenancy mirrors basic-gin's .pathwise.yaml.
func corpusTenancy() *config.Config {
	return &config.Config{Tenancy: config.Tenancy{Column: "org_id", Tables: []string{"students", "batches", "users"}}}
}

func reached(t *testing.T, sql string) analyzer.ReachedQuery {
	t.Helper()
	q, err := dataflow.Parse(sql)
	if err != nil {
		t.Fatalf("parse %q: %v", sql, err)
	}
	return analyzer.ReachedQuery{SQL: sql, Query: q}
}

func TestTenantFilter(t *testing.T) {
	cfg := corpusTenancy()

	// allBatches: unscoped read of a tenant table → must fire.
	fs := findings.TenantFilter("x", "GET /batches", []analyzer.ReachedQuery{
		reached(t, "SELECT id, org_id, name FROM batches ORDER BY name"),
	}, cfg)
	if len(fs) != 1 || fs[0].Kind != model.FindingMissingTenantFilter {
		t.Fatalf("allBatches: findings = %+v, want one missing_tenant_filter", fs)
	}

	// reportRows: scoped on org_id → must not fire.
	if fs := findings.TenantFilter("x", "GET /dashboard", []analyzer.ReachedQuery{
		reached(t, "SELECT id, org_id, name FROM batches WHERE org_id = $1"),
	}, cfg); len(fs) != 0 {
		t.Errorf("reportRows (scoped by org_id) should not fire: %+v", fs)
	}

	// studentByID: fetch by key on a tenant table → the ownership finding's
	// concern, not tenant; the equality predicate suppresses it here.
	if fs := findings.TenantFilter("x", "GET /students/{id}", []analyzer.ReachedQuery{
		reached(t, "SELECT id, org_id, batch_id, name, email FROM students WHERE id = $1"),
	}, cfg); len(fs) != 0 {
		t.Errorf("studentByID (fetch by key) should not fire tenant filter: %+v", fs)
	}

	// lookupSession: filtered by token → suppressed.
	if fs := findings.TenantFilter("x", "POST /login", []analyzer.ReachedQuery{
		reached(t, "SELECT u.id, u.org_id, u.role FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.token = $1 AND s.expires_at > now()"),
	}, cfg); len(fs) != 0 {
		t.Errorf("session lookup should not fire tenant filter: %+v", fs)
	}

	// No config → inactive, never fires.
	if fs := findings.TenantFilter("x", "GET /batches", []analyzer.ReachedQuery{
		reached(t, "SELECT id FROM batches"),
	}, &config.Config{}); len(fs) != 0 {
		t.Errorf("without tenancy config the check is inactive: %+v", fs)
	}
}
