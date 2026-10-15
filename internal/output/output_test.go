package output_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/YarKhan02/PathWise/internal/model"
	"github.com/YarKhan02/PathWise/internal/output"
)

func sample() []model.Finding {
	return []model.Finding{
		{
			Kind: model.FindingMissingOwnershipCheck, Severity: model.SeverityHigh, Confidence: model.ConfidenceProven,
			Endpoint: "GET /students/{id}", Title: "Missing ownership check (IDOR)", Detail: "request value reaches a key predicate",
			Locations: []model.SourceLocation{{File: "store.go", StartLine: 47}},
		},
		{
			Kind: model.FindingResponseFieldExceedsScope, Severity: model.SeverityMedium, Confidence: model.ConfidenceLikely,
			Endpoint: "GET /students/{id}/profile", Title: "Response exposes out-of-scope field", Detail: "national_id",
		},
	}
}

func TestJSON(t *testing.T) {
	b, err := output.JSON("demo", "s1", sample())
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	var r output.Report
	if err := json.Unmarshal(b, &r); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if r.Project != "demo" || len(r.Findings) != 2 || r.Findings[0].Endpoint != "GET /students/{id}" {
		t.Errorf("report = %+v", r)
	}
	// nil findings must serialize as [], not null.
	b2, _ := output.JSON("demo", "s1", nil)
	if strings.Contains(string(b2), "null") {
		t.Errorf("empty findings must be []: %s", b2)
	}
}

func TestSARIF(t *testing.T) {
	b, err := output.SARIF("demo", "s1", sample())
	if err != nil {
		t.Fatalf("SARIF: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc["version"] != "2.1.0" {
		t.Errorf("version = %v, want 2.1.0", doc["version"])
	}
	s := string(b)
	for _, want := range []string{`"name": "PathWise"`, `"ruleId": "missing_ownership_check"`, `"level": "error"`, `"level": "warning"`, `"startLine": 47`} {
		if !strings.Contains(s, want) {
			t.Errorf("SARIF missing %q", want)
		}
	}
}
