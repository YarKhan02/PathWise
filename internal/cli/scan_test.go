package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/YarKhan02/PathWise/internal/config"
	"github.com/YarKhan02/PathWise/internal/model"
)

// TestDiscoverRoutesReportsBuildError pins the landmine behavior: when the
// target does not type-check, route discovery must record a build_error fact
// and say so, never return an empty result that reads as "no routes found".
// testdata/broken is a module with a deliberate type error.
func TestDiscoverRoutesReportsBuildError(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("testdata", "broken"))
	if err != nil {
		t.Fatalf("resolve fixture: %v", err)
	}

	ca := analyzeCode("scan_test", dir, &config.Config{}, nil, false)

	found := false
	for _, f := range ca.facts {
		if f.Evidence.Status == model.StatusUnverified && f.Evidence.UnresolvedReason == model.ReasonBuildError {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a build_error fact, got %d facts with summary %q", len(ca.facts), ca.routeSummary)
	}
	if !strings.Contains(ca.routeSummary, "build error") {
		t.Errorf("summary %q should name the build failure", ca.routeSummary)
	}
}
