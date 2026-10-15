package model

// EvidenceStatus records how strongly a fact is supported.
//
// The four statuses are deliberately not a confidence ladder. code_confirmed
// and runtime_observed are different *kinds* of evidence, not different
// strengths of the same kind, and they are never merged: a static conclusion
// and a runtime observation about the same endpoint are stored separately so
// that a contradiction between them stays visible. That contradiction is the
// single highest-value signal PathWise can emit.
type EvidenceStatus string

const (
	// StatusCodeConfirmed means the analyzer proved this from source with
	// type information and control flow, not from a name match.
	StatusCodeConfirmed EvidenceStatus = "code_confirmed"

	// StatusRuntimeObserved means a live request demonstrated this. It is
	// scoped to the scenario that produced it and generalizes to nothing.
	StatusRuntimeObserved EvidenceStatus = "runtime_observed"

	// StatusInferred means the analyzer applied a heuristic that may be
	// wrong, e.g. treating a column named org_id as a tenant discriminator.
	StatusInferred EvidenceStatus = "inferred"

	// StatusUnverified means analysis could not reach a conclusion. It must
	// always carry an UnresolvedReason; "unverified" alone is noise a
	// developer cannot act on.
	StatusUnverified EvidenceStatus = "unverified"
)

// UnresolvedReason explains why analysis stopped. This exists because
// "unverified" on its own is indistinguishable from "safe" to a reader, which
// is the most dangerous failure mode the tool has. Every unresolved fact names
// the specific thing that defeated it.
type UnresolvedReason string

const (
	ReasonNone UnresolvedReason = ""

	// ReasonInterfaceDispatch is a call through an interface whose concrete
	// target could not be narrowed to one function. Attrs should carry the
	// candidate count and the interface's qualified name.
	ReasonInterfaceDispatch UnresolvedReason = "interface_dispatch"

	// ReasonDynamicSQL is query text assembled at runtime (fmt.Sprintf,
	// string concatenation, a query builder). The predicate set is unknown,
	// so no filter conclusion may be drawn.
	ReasonDynamicSQL UnresolvedReason = "dynamic_sql"

	// ReasonDynamicRoute is a route pattern that is not a constant string.
	ReasonDynamicRoute UnresolvedReason = "dynamic_route"

	// ReasonReflection is access through reflect or unsafe. VTA is sound only
	// modulo these, so they are recorded rather than silently ignored.
	ReasonReflection UnresolvedReason = "reflection"

	// ReasonExternalPackage is a call leaving the analyzed module into a
	// dependency whose body was not analyzed.
	ReasonExternalPackage UnresolvedReason = "external_package"

	// ReasonBuildError means go/packages could not type-check the target.
	// This is the most common real-world failure and must never be reported
	// as "no findings".
	ReasonBuildError UnresolvedReason = "build_error"

	// ReasonMissingConfig means the check requires configuration the project
	// has not supplied, e.g. no tenant column declared.
	ReasonMissingConfig UnresolvedReason = "missing_config"

	// ReasonUnsupportedQueryLayer is a data-access call through a layer this
	// analyzer version does not understand (GORM, ent, a custom builder).
	ReasonUnsupportedQueryLayer UnresolvedReason = "unsupported_query_layer"

	// ReasonNoResponseSchema means response fields could not be determined
	// because the handler serializes a value of indeterminate shape.
	ReasonNoResponseSchema UnresolvedReason = "no_response_schema"
)

// SourceLocation points at the code backing a fact. Every code_confirmed fact
// carries at least one, because a finding a developer cannot navigate to is a
// finding they will not fix.
type SourceLocation struct {
	File      string `json:"file"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// Evidence is attached to every fact, node and edge.
type Evidence struct {
	Status           EvidenceStatus   `json:"status"`
	UnresolvedReason UnresolvedReason `json:"unresolved_reason,omitempty"`
	Source           []SourceLocation `json:"source,omitempty"`
	Details          string           `json:"details,omitempty"`

	// ScenarioID links runtime evidence to the exact test that produced it.
	// Required when Status is StatusRuntimeObserved and empty otherwise.
	ScenarioID string `json:"scenario_id,omitempty"`
}

// Valid reports whether the evidence is internally consistent. Enforcing this
// at construction time prevents the two mistakes that would quietly destroy
// the tool's credibility: an unverified fact with no stated reason, and
// runtime evidence not tied to a scenario.
func (e Evidence) Valid() bool {
	switch e.Status {
	case StatusUnverified:
		return e.UnresolvedReason != ReasonNone
	case StatusRuntimeObserved:
		return e.ScenarioID != ""
	case StatusCodeConfirmed:
		return len(e.Source) > 0
	case StatusInferred:
		return e.Details != ""
	default:
		return false
	}
}

// CodeConfirmed builds evidence proven from source.
func CodeConfirmed(details string, locs ...SourceLocation) Evidence {
	return Evidence{Status: StatusCodeConfirmed, Source: locs, Details: details}
}

// Inferred builds heuristic evidence. The details must state which heuristic
// fired so a reader can judge it.
func Inferred(heuristic string, locs ...SourceLocation) Evidence {
	return Evidence{Status: StatusInferred, Source: locs, Details: heuristic}
}

// Unresolved builds evidence for analysis that gave up, naming why.
func Unresolved(reason UnresolvedReason, details string, locs ...SourceLocation) Evidence {
	return Evidence{
		Status:           StatusUnverified,
		UnresolvedReason: reason,
		Source:           locs,
		Details:          details,
	}
}
