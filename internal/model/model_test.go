package model

import "testing"

func TestNormalizeRoute(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"/students/:id", "/students/{id}"},
		{"/students/{id}", "/students/{id}"},
		{"students/:id", "/students/{id}"},
		{"/students/:id/", "/students/{id}"},
		{"/files/*filepath", "/files/{*}"},
		{"/", "/"},
		{"", "/"},
	}

	for _, tc := range tests {
		if got := NormalizeRoute(tc.in); got != tc.want {
			t.Errorf("NormalizeRoute(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestRouteSubjectIsFrameworkAgnostic is the property that makes git diffing
// possible: the same logical route must have one identity no matter which
// framework's syntax declared it.
func TestRouteSubjectIsFrameworkAgnostic(t *testing.T) {
	gin := RouteSubject("get", "/students/:id")
	chi := RouteSubject("GET", "/students/{id}")

	if gin != chi {
		t.Errorf("gin %q and chi %q should share one identity", gin, chi)
	}
	if want := "GET /students/{id}"; gin != want {
		t.Errorf("subject = %q, want %q", gin, want)
	}
}

func TestEvidenceValid(t *testing.T) {
	tests := []struct {
		name string
		ev   Evidence
		want bool
	}{
		{
			name: "unverified without reason is rejected",
			ev:   Evidence{Status: StatusUnverified},
			want: false,
		},
		{
			name: "unverified with reason is accepted",
			ev:   Unresolved(ReasonDynamicSQL, "query built with fmt.Sprintf"),
			want: true,
		},
		{
			name: "runtime without scenario is rejected",
			ev:   Evidence{Status: StatusRuntimeObserved},
			want: false,
		},
		{
			name: "runtime with scenario is accepted",
			ev:   Evidence{Status: StatusRuntimeObserved, ScenarioID: "scn_1"},
			want: true,
		},
		{
			name: "code confirmed without location is rejected",
			ev:   Evidence{Status: StatusCodeConfirmed, Details: "x"},
			want: false,
		},
		{
			name: "code confirmed with location is accepted",
			ev:   CodeConfirmed("abort dominates response", SourceLocation{File: "h.go", StartLine: 1, EndLine: 2}),
			want: true,
		},
		{
			name: "inferred without stated heuristic is rejected",
			ev:   Evidence{Status: StatusInferred},
			want: false,
		},
		{
			name: "unknown status is rejected",
			ev:   Evidence{Status: "wishful"},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.ev.Valid(); got != tc.want {
				t.Errorf("Valid() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFingerprintStability(t *testing.T) {
	a := Fact{
		Kind:     FactRoute,
		Subject:  "GET /students/{id}",
		Attrs:    map[string]any{"handler": "main.GetStudent", "framework": "gin"},
		Evidence: CodeConfirmed("x", SourceLocation{File: "main.go", StartLine: 10, EndLine: 10}),
	}
	// Same content, attrs declared in a different order.
	b := Fact{
		Kind:     FactRoute,
		Subject:  "GET /students/{id}",
		Attrs:    map[string]any{"framework": "gin", "handler": "main.GetStudent"},
		Evidence: CodeConfirmed("y", SourceLocation{File: "main.go", StartLine: 99, EndLine: 99}),
	}

	if a.Fingerprint() != b.Fingerprint() {
		t.Error("fingerprint must not depend on attr ordering or source position")
	}

	c := a
	c.Subject = "GET /students/{id}/safe"
	if a.Fingerprint() == c.Fingerprint() {
		t.Error("different subjects must fingerprint differently")
	}
}

// TestRequiresConfig pins which checks need project policy. Getting this
// wrong means reporting a clean scan for a check that never ran.
func TestRequiresConfig(t *testing.T) {
	want := map[FindingKind]bool{
		FindingMissingOwnershipCheck:     false,
		FindingRoleCheckUnenforced:       false,
		FindingMissingTenantFilter:       true,
		FindingResponseFieldExceedsScope: true,
	}

	for kind, expect := range want {
		if got := kind.RequiresConfig(); got != expect {
			t.Errorf("%s.RequiresConfig() = %v, want %v", kind, got, expect)
		}
	}

	if got, want := len(AllFindingKinds()), len(want); got != want {
		t.Errorf("AllFindingKinds() has %d entries, want %d: the taxonomy and this test disagree", got, want)
	}
}
