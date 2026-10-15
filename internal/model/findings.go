package model

// FindingKind is the fixed finding taxonomy. It is fixed on purpose: each
// kind states exactly what the analyzer must prove, and the analysis is built
// backward from these four. Adding a fifth kind is a deliberate decision that
// comes with a new proof obligation, not an incidental discovery.
type FindingKind string

const (
	// FindingMissingOwnershipCheck fires when a request-controlled value
	// reaches a primary-key or unique-key predicate, no identity-derived
	// predicate constrains the same query, and no ownership check dominates
	// every path from the handler entry to the response write.
	//
	// This is the classic IDOR. Zero configuration required.
	FindingMissingOwnershipCheck FindingKind = "missing_ownership_check"

	// FindingMissingTenantFilter fires when a query touches a table declared
	// tenant-scoped and carries no predicate on that table's tenant column.
	//
	// Requires configuration: PathWise cannot know which column discriminates
	// tenants. Without config this emits an unverified fact naming
	// ReasonMissingConfig, never a finding.
	FindingMissingTenantFilter FindingKind = "missing_tenant_filter"

	// FindingRoleCheckUnenforced fires when a call classified as a role or
	// permission check has its result discarded, or when its failure branch
	// does not dominate the response write — the check runs and the request
	// proceeds regardless.
	//
	// Zero configuration required.
	FindingRoleCheckUnenforced FindingKind = "role_check_unenforced"

	// FindingResponseFieldExceedsScope fires when a response serializes a
	// field backed by a column declared sensitive, on an endpoint whose
	// declared scope does not include it.
	//
	// Requires configuration: "sensitive" and "in scope" are project
	// policy, not properties the analyzer can derive.
	FindingResponseFieldExceedsScope FindingKind = "response_field_exceeds_scope"
)

// RequiresConfig reports whether this finding kind can fire with no
// .pathwise.yaml present.
//
// Two of the four kinds need project policy to mean anything. The CLI uses
// this to tell a user running without config which checks are inactive,
// rather than reporting a clean scan that was in fact a partial one.
func (k FindingKind) RequiresConfig() bool {
	switch k {
	case FindingMissingTenantFilter, FindingResponseFieldExceedsScope:
		return true
	default:
		return false
	}
}

// AllFindingKinds is the complete taxonomy, in reporting order.
func AllFindingKinds() []FindingKind {
	return []FindingKind{
		FindingMissingOwnershipCheck,
		FindingRoleCheckUnenforced,
		FindingMissingTenantFilter,
		FindingResponseFieldExceedsScope,
	}
}

// Severity is the finding's assessed impact.
type Severity string

const (
	SeverityHigh   Severity = "high"
	SeverityMedium Severity = "medium"
	SeverityLow    Severity = "low"
	SeverityInfo   Severity = "info"
)

// Confidence is how sure the analyzer is that the finding is real. It is
// reported separately from severity because a high-impact guess and a
// high-impact proof demand different responses from a developer.
type Confidence string

const (
	// ConfidenceProven means every premise is code_confirmed.
	ConfidenceProven Confidence = "proven"

	// ConfidenceLikely means the premises hold but at least one rests on a
	// heuristic.
	ConfidenceLikely Confidence = "likely"

	// ConfidencePossible means a premise is unresolved and the finding may
	// be a false positive.
	ConfidencePossible Confidence = "possible"
)

// Finding is a reported authorization discrepancy.
type Finding struct {
	ID     string `json:"id"`
	ScanID string `json:"scan_id"`

	Kind       FindingKind `json:"kind"`
	Severity   Severity    `json:"severity"`
	Confidence Confidence  `json:"confidence"`

	// Endpoint is the normalized route this finding concerns.
	Endpoint string `json:"endpoint"`

	// StableKey identifies the same finding across scans so that the diff
	// engine can report introduced, resolved and no-longer-verifiable
	// findings rather than a churn of new IDs.
	StableKey string `json:"stable_key"`

	Title  string `json:"title"`
	Detail string `json:"detail"`
	Remedy string `json:"remedy,omitempty"`

	// Locations are the code sites a developer should open first.
	Locations []SourceLocation `json:"locations,omitempty"`

	// Premises records the facts this finding was derived from, so a reader
	// can audit the chain rather than trusting the conclusion.
	Premises []string `json:"premises,omitempty"`

	// RuntimeDivergence is set when runtime evidence contradicts the static
	// conclusion. A static "protected" that a live request defeated is the
	// highest-confidence output PathWise produces.
	RuntimeDivergence bool `json:"runtime_divergence,omitempty"`
}
