package findings

import (
	"fmt"

	"github.com/YarKhan02/PathWise/internal/analyzer"
	"github.com/YarKhan02/PathWise/internal/model"
)

// OwnershipCheck reports the classic IDOR: a request-controlled value reaches a
// primary- or unique-key predicate, no identity-derived predicate constrains the
// same query, and no ownership check dominates the path to the response.
//
// It is active only when the schema is known (keyCols non-empty) — "primary or
// unique key" is a schema fact, so this finding needs a live database. It is
// gated to authenticated endpoints (authenticated) and to queries reachable from
// the handler itself (queries), so the public login's unique-email lookup and
// the auth middleware's session-token lookup never false-positive. hasGuard is
// true when the handler aborts on an identity-derived branch that dominates the
// response (authorization.HasOwnershipGuard).
// schemaConfirmed is whether the keys come from a live database (confirmed) or
// a reconstructed-from-migrations schema (inferred); it sets the finding's
// confidence, because the primary-key premise is only as strong as its source.
func OwnershipCheck(scanID, endpoint string, authenticated, hasGuard bool,
	queries []analyzer.ReachedQuery, keyCols map[string]map[string]bool, cls *analyzer.Classifier, schemaConfirmed bool) []model.Finding {

	if !authenticated || hasGuard || len(keyCols) == 0 || cls == nil {
		return nil
	}

	for _, rq := range queries {
		q := rq.Query
		if q == nil {
			continue
		}
		// Clause: no identity-derived predicate scopes this query to the caller.
		if hasIdentityPredicate(rq, cls) {
			continue
		}
		for _, pred := range q.EqPredicates {
			if pred.Param <= 0 || pred.Param > len(rq.Params) {
				continue
			}
			if !isKeyColumn(pred.Column, q.Tables, keyCols) {
				continue
			}
			if cls.Provenance(rq.Params[pred.Param-1]) != analyzer.ProvRequest {
				continue
			}
			return []model.Finding{idorFinding(scanID, endpoint, pred.Column, rq, schemaConfirmed)}
		}
	}
	return nil
}

// hasIdentityPredicate reports whether any equality predicate binds a value
// derived from the authenticated principal — i.e. the query is scoped to the
// caller's own rows.
func hasIdentityPredicate(rq analyzer.ReachedQuery, cls *analyzer.Classifier) bool {
	for _, pred := range rq.Query.EqPredicates {
		if pred.Param > 0 && pred.Param <= len(rq.Params) &&
			cls.Provenance(rq.Params[pred.Param-1]) == analyzer.ProvIdentity {
			return true
		}
	}
	return false
}

// isKeyColumn reports whether column is a primary- or unique-key column of any
// table the query touches.
func isKeyColumn(column string, tables []string, keyCols map[string]map[string]bool) bool {
	for _, t := range tables {
		if keyCols[t][column] {
			return true
		}
	}
	return false
}

func idorFinding(scanID, endpoint, column string, rq analyzer.ReachedQuery, schemaConfirmed bool) model.Finding {
	confidence := model.ConfidenceLikely // the key premise rests on a reconstructed schema
	if schemaConfirmed {
		confidence = model.ConfidenceProven
	}
	return model.Finding{
		ScanID:     scanID,
		Kind:       model.FindingMissingOwnershipCheck,
		Severity:   model.SeverityHigh,
		Confidence: confidence,
		Endpoint:   endpoint,
		StableKey:  string(model.FindingMissingOwnershipCheck) + "|" + endpoint + "|" + column,
		Title:      "Missing ownership check (IDOR)",
		Detail: fmt.Sprintf(
			"A request-controlled value reaches the key predicate %q, no identity-derived predicate constrains the query, and no ownership check dominates the response. Any authenticated caller can read another user's row.",
			column),
		Remedy:    "Constrain the query by the authenticated principal (an identity-derived predicate), or add an ownership check that aborts before the response on every path.",
		Locations: []model.SourceLocation{{File: rq.Pos.Filename, StartLine: rq.Pos.Line, EndLine: rq.Pos.Line}},
		Premises:  []string{"query: " + normalizeSQL(rq.SQL)},
	}
}
