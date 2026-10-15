// Package output renders findings in machine-readable formats for CI and
// tooling. JSON is the findings as stored; SARIF is the CI standard that gets
// GitHub code-scanning integration almost for free. The findings are reported
// as-is — no format invents, reclassifies or omits one.
package output

import (
	"encoding/json"

	"github.com/YarKhan02/PathWise/internal/model"
)

// Report is the JSON envelope: the scan it came from and its findings.
type Report struct {
	Project  string          `json:"project"`
	ScanID   string          `json:"scanId"`
	Findings []model.Finding `json:"findings"`
}

// JSON renders findings as an indented JSON report.
func JSON(project, scanID string, findings []model.Finding) ([]byte, error) {
	if findings == nil {
		findings = []model.Finding{}
	}
	return json.MarshalIndent(Report{Project: project, ScanID: scanID, Findings: findings}, "", "  ")
}

// SARIF renders findings as SARIF 2.1.0, the static-analysis interchange format
// GitHub code scanning and many editors consume.
func SARIF(project, scanID string, findings []model.Finding) ([]byte, error) {
	rules := map[string]sarifRule{}
	results := make([]sarifResult, 0, len(findings))

	for _, f := range findings {
		kind := string(f.Kind)
		if _, ok := rules[kind]; !ok {
			rules[kind] = sarifRule{
				ID:               kind,
				Name:             kind,
				ShortDescription: sarifText{Text: f.Title},
			}
		}
		var locs []sarifLocation
		for _, l := range f.Locations {
			locs = append(locs, sarifLocation{PhysicalLocation: sarifPhysical{
				ArtifactLocation: sarifArtifact{URI: l.File},
				Region:           sarifRegion{StartLine: max(l.StartLine, 1)},
			}})
		}
		results = append(results, sarifResult{
			RuleID:    kind,
			Level:     sarifLevel(f.Severity),
			Message:   sarifText{Text: f.Title + ": " + f.Detail},
			Locations: locs,
			Properties: map[string]string{
				"confidence": string(f.Confidence),
				"endpoint":   f.Endpoint,
				"severity":   string(f.Severity),
			},
		})
	}

	ruleList := make([]sarifRule, 0, len(rules))
	for _, r := range rules {
		ruleList = append(ruleList, r)
	}

	doc := sarifLog{
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name:           "PathWise",
				InformationURI: "https://github.com/YarKhan02/PathWise",
				Rules:          ruleList,
			}},
			Results:    results,
			Properties: map[string]string{"project": project, "scanId": scanID},
		}},
	}
	return json.MarshalIndent(doc, "", "  ")
}

// sarifLevel maps a finding severity to a SARIF result level.
func sarifLevel(sev model.Severity) string {
	switch sev {
	case model.SeverityHigh:
		return "error"
	case model.SeverityMedium:
		return "warning"
	default:
		return "note"
	}
}

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}
type sarifRun struct {
	Tool       sarifTool         `json:"tool"`
	Results    []sarifResult     `json:"results"`
	Properties map[string]string `json:"properties,omitempty"`
}
type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}
type sarifDriver struct {
	Name           string      `json:"name"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}
type sarifRule struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	ShortDescription sarifText `json:"shortDescription"`
}
type sarifResult struct {
	RuleID     string            `json:"ruleId"`
	Level      string            `json:"level"`
	Message    sarifText         `json:"message"`
	Locations  []sarifLocation   `json:"locations,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}
type sarifText struct {
	Text string `json:"text"`
}
type sarifLocation struct {
	PhysicalLocation sarifPhysical `json:"physicalLocation"`
}
type sarifPhysical struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
	Region           sarifRegion   `json:"region"`
}
type sarifArtifact struct {
	URI string `json:"uri"`
}
type sarifRegion struct {
	StartLine int `json:"startLine"`
}
