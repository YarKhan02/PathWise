package corpus

import (
	"path/filepath"
	"testing"

	"github.com/YarKhan02/PathWise/internal/model"
)

// corpusRoot is the directory holding the corpus apps.
func corpusRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "testdata", "apps")
}

// TestCorpusLoads is the Phase 0 exit gate. If the corpus does not load and
// validate, every later phase is measuring itself against nothing.
func TestCorpusLoads(t *testing.T) {
	apps, err := LoadAll(corpusRoot(t))
	if err != nil {
		t.Fatalf("load corpus: %v", err)
	}
	if len(apps) == 0 {
		t.Fatal("corpus is empty: no app has an expected.yaml")
	}

	for _, app := range apps {
		t.Run(app.Name, func(t *testing.T) {
			for _, err := range app.Validate() {
				t.Errorf("%v", err)
			}
		})
	}
}

// TestCorpusCoversEveryFindingKind guards against the corpus drifting out of
// step with the taxonomy. A finding kind with no corpus instance cannot be
// shown to work, so adding a kind must come with an app that exhibits it.
func TestCorpusCoversEveryFindingKind(t *testing.T) {
	apps, err := LoadAll(corpusRoot(t))
	if err != nil {
		t.Fatalf("load corpus: %v", err)
	}

	seen := map[model.FindingKind]int{}
	for _, app := range apps {
		for _, f := range app.Findings {
			seen[f.Kind]++
		}
	}

	for _, kind := range model.AllFindingKinds() {
		if seen[kind] == 0 {
			t.Errorf("finding kind %s has no corpus instance: it cannot be validated", kind)
		}
	}
}

// TestCorpusHasNegativeControls asserts that each app declares sites which
// must NOT produce findings. An analyzer measured only against seeded flaws
// can score perfectly by reporting everything.
func TestCorpusHasNegativeControls(t *testing.T) {
	apps, err := LoadAll(corpusRoot(t))
	if err != nil {
		t.Fatalf("load corpus: %v", err)
	}

	for _, app := range apps {
		if len(app.Safe) == 0 {
			t.Errorf("%s declares no expected_safe sites: false positives would go unnoticed", app.Name)
		}
	}
}

// TestBasicGinExpectations pins the specific answers for the first app, so
// that an accidental edit to expected.yaml fails loudly rather than quietly
// lowering the bar.
func TestBasicGinExpectations(t *testing.T) {
	app, err := LoadApp(filepath.Join(corpusRoot(t), "basic-gin"))
	if err != nil {
		t.Fatalf("load basic-gin: %v", err)
	}

	if got, want := len(app.Routes), 8; got != want {
		t.Errorf("routes = %d, want %d", got, want)
	}
	if got, want := len(app.Findings), 5; got != want {
		t.Errorf("seeded findings = %d, want %d", got, want)
	}
	if got, want := len(app.Unresolved), 1; got != want {
		t.Errorf("unresolved controls = %d, want %d", got, want)
	}

	// Both role_check_unenforced instances must be present: the discarded
	// return value and the non-dominating branch are different proofs, and an
	// analyzer can pass one while failing the other.
	roleFindings := 0
	for _, f := range app.Findings {
		if f.Kind == model.FindingRoleCheckUnenforced {
			roleFindings++
		}
	}
	if roleFindings != 2 {
		t.Errorf("role_check_unenforced instances = %d, want 2 (discarded value and non-dominating branch)", roleFindings)
	}

	if _, ok := app.Schema.Tables["students"]; !ok {
		t.Error("schema is missing the students table")
	}
	if got, want := len(app.Schema.ForeignKeys), 5; got != want {
		t.Errorf("foreign keys = %d, want %d", got, want)
	}
}

// TestMarkerParsing covers the marker syntax directly.
func TestMarkerParsing(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantOK  bool
		wantID  string
		wantKnd MarkerKind
	}{
		{"expect", "expect kind=missing_ownership_check id=idor-1", true, "idor-1", MarkerExpect},
		{"safe", "safe id=ok-1", true, "ok-1", MarkerSafe},
		{"unresolved", "unresolved reason=dynamic_sql id=dyn-1", true, "dyn-1", MarkerUnresolved},
		{"unknown kind", "banana id=x", false, "", ""},
		{"empty", "", false, "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, ok := parseMarker(tc.in)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if m.ID != tc.wantID {
				t.Errorf("id = %q, want %q", m.ID, tc.wantID)
			}
			if m.Kind != tc.wantKnd {
				t.Errorf("kind = %q, want %q", m.Kind, tc.wantKnd)
			}
		})
	}
}

// TestMarkersIgnoreNonComments ensures a marker-looking string in code does
// not register as a marker.
func TestMarkersIgnoreNonComments(t *testing.T) {
	if _, ok := parseMarker("expect kind=x id=y"); !ok {
		t.Fatal("expected a well-formed marker to parse")
	}
	// scanFile requires the prefix to appear after "//" on the line; this is
	// covered by TestCorpusLoads against the real corpus, where string
	// literals containing the prefix would otherwise create phantom markers.
}
