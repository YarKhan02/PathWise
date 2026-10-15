// Package findings turns the analyzed facts into the four authorization
// findings. Each rule states the proof it discharges and cites the query or
// code it rests on, so a reader can audit the conclusion rather than trust it.
package findings

import (
	"fmt"
	"strings"

	"github.com/YarKhan02/PathWise/internal/analyzer"
	"github.com/YarKhan02/PathWise/internal/config"
	"github.com/YarKhan02/PathWise/internal/model"
)

// TenantFilter reports a read of a tenant-scoped table that carries no predicate
// on the tenant column, so it can return rows belonging to other tenants.
//
// It fires only when configured (which table is tenant-scoped, and on which
// column, is project policy). A read is flagged when the tenant column is absent
// from its WHERE clause and the read is not narrowed by an equality predicate —
// an unfiltered or range-filtered list, the shape that leaks across tenants.
//
// shortcut: "narrowed by an equality predicate" stands in for "fetches a single
// row by key". The precise test is a primary/unique-key equality, which needs
// the schema; until the schema is wired into this rule, an equality on any
// column suppresses the finding, which can miss a broad read filtered on a
// non-key column. It does not produce false positives.
func TenantFilter(scanID, endpoint string, queries []analyzer.ReachedQuery, cfg *config.Config) []model.Finding {
	if !cfg.TenancyConfigured() {
		return nil
	}
	col := strings.ToLower(cfg.Tenancy.Column)
	tenant := map[string]bool{}
	for _, t := range cfg.Tenancy.Tables {
		tenant[strings.ToLower(t)] = true
	}

	var out []model.Finding
	seen := map[string]bool{}
	for _, rq := range queries {
		q := rq.Query
		if q == nil || q.Operation != "SELECT" {
			continue
		}
		if q.HasWhereColumn(col) || len(q.EqPredicates) > 0 {
			continue // scoped by the tenant column, or narrowed to specific rows
		}
		for _, t := range q.Tables {
			if !tenant[t] {
				continue
			}
			key := endpoint + "|" + t
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, model.Finding{
				ScanID:     scanID,
				Kind:       model.FindingMissingTenantFilter,
				Severity:   model.SeverityHigh,
				Confidence: model.ConfidenceProven,
				Endpoint:   endpoint,
				StableKey:  string(model.FindingMissingTenantFilter) + "|" + endpoint + "|" + t,
				Title:      "Missing tenant filter on " + t,
				Detail: fmt.Sprintf(
					"A query reachable from this endpoint reads %s, declared tenant-scoped, with no predicate on the tenant column %q — it can return rows from other tenants.",
					t, cfg.Tenancy.Column),
				Remedy:    fmt.Sprintf("Constrain the query with %s.%s bound to the authenticated principal's tenant.", t, cfg.Tenancy.Column),
				Locations: []model.SourceLocation{{File: rq.Pos.Filename, StartLine: rq.Pos.Line, EndLine: rq.Pos.Line}},
				Premises:  []string{"query: " + normalizeSQL(rq.SQL)},
			})
		}
	}
	return out
}

// normalizeSQL collapses a multi-line query to one line for a compact premise.
func normalizeSQL(sql string) string {
	return strings.Join(strings.Fields(sql), " ")
}
