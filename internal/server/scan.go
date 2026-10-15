// Package server hosts the interactive authorization dashboard over net/http.
// The single-page app and its assets are embedded, so the dashboard ships
// inside the pathwise binary with no network dependency. /api/scan serves the
// full projection of the latest scan as JSON; the page renders it.
package server

import (
	"sort"

	"github.com/YarKhan02/pathwise/internal/model"
	"github.com/YarKhan02/pathwise/internal/report"
)

// Scan is the JSON contract the dashboard consumes. It is a projection of a
// scan's facts and findings, shaped for the UI — the analyzer's conclusions,
// not a parallel schema.
type Scan struct {
	Project      string          `json:"project"`
	ScanID       string          `json:"scanId"`
	SchemaSource string          `json:"schemaSource"`
	GeneratedAt  string          `json:"generatedAt"`
	Counts       Counts          `json:"counts"`
	Roles        []string        `json:"roles"`
	Tables       []TableInfo     `json:"tables"`
	Endpoints    []Endpoint      `json:"endpoints"`
	Findings     []model.Finding `json:"findings"`
	// Runtime verification, when a `pathwise test` run was recorded against this
	// scan. Empty when none ran — the dashboard then says so rather than implying
	// the endpoints are untested-and-safe. These are stored apart from the static
	// findings and never merged with them.
	RuntimeTests    []RuntimeTest `json:"runtimeTests"`
	RuntimeTestedAt string        `json:"runtimeTestedAt,omitempty"`
}

// RuntimeTest is one scenario's observed outcome, projected for the dashboard.
type RuntimeTest struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Endpoint    string `json:"endpoint"`
	Role        string `json:"role"`
	Expectation string `json:"expectation"` // allowed | blocked (the static prediction)
	Outcome     string `json:"outcome"`     // pass | fail | blocked | unverified | divergent
	Status      int    `json:"status,omitempty"`
	Detail      string `json:"detail"`
}

// Counts are the headline numbers for the overview.
type Counts struct {
	Endpoints  int            `json:"endpoints"`
	Roles      int            `json:"roles"`
	Tables     int            `json:"tables"`
	Findings   int            `json:"findings"`
	Unresolved int            `json:"unresolved"`
	Severity   map[string]int `json:"severity"`
}

// Endpoint is one API route and everything the scan established about it.
type Endpoint struct {
	ID          string          `json:"id"`
	Method      string          `json:"method"`
	Path        string          `json:"path"`
	Handler     string          `json:"handler"`
	HandlerFile string          `json:"handlerFile,omitempty"`
	HandlerLine int             `json:"handlerLine,omitempty"`
	Roles       []string        `json:"roles"`
	Middleware  []Middleware    `json:"middleware"`
	Checks      []Check         `json:"checks"`
	Tables      []TableRef      `json:"tables"`
	Unresolved  bool            `json:"unresolved"`
	Findings    []model.Finding `json:"findings"`
}

// Middleware is one gate in an endpoint's chain; Args carries string literals
// such as the role in RequireRole("admin").
type Middleware struct {
	Name string   `json:"name"`
	Args []string `json:"args,omitempty"`
}

// Check is an authorization-check call in the handler and whether it gates the
// response.
type Check struct {
	Func        string `json:"func"`
	Enforcement string `json:"enforcement"`
	Enforced    bool   `json:"enforced"`
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
}

// TableRef is a table an endpoint reaches with the operations it performs.
type TableRef struct {
	Table      string   `json:"table"`
	Operations []string `json:"operations"`
}

// TableInfo is a table and the endpoints that reach it — the reverse edges for
// the database view.
type TableInfo struct {
	Name       string   `json:"name"`
	Endpoints  []string `json:"endpoints"`
	Operations []string `json:"operations"`
}

// BuildScan projects a report into the dashboard contract.
func BuildScan(r *report.Report) Scan {
	s := Scan{
		Project:      r.Project,
		ScanID:       r.ScanID,
		SchemaSource: r.SchemaSource,
		GeneratedAt:  r.GeneratedAt,
		Roles:        nonNil(r.Roles),
		Findings:     []model.Finding{},
		Endpoints:    []Endpoint{},
		Tables:       []TableInfo{},
		RuntimeTests: []RuntimeTest{},
		Counts: Counts{
			Endpoints: r.RouteCount, Roles: r.RoleCount, Tables: r.TableCount,
			Findings: r.FindingCount, Unresolved: r.UnresolvedCount,
			Severity: map[string]int{},
		},
	}

	tableEndpoints := map[string]map[string]bool{}
	tableOps := map[string]map[string]bool{}

	for _, e := range r.Endpoints {
		ep := Endpoint{
			ID: e.Subject, Method: e.Method, Path: e.Path, Handler: e.Handler,
			HandlerFile: e.HandlerFile, HandlerLine: e.HandlerLine,
			Roles: nonNil(e.Roles), Unresolved: e.Unresolved,
			Middleware: []Middleware{}, Checks: []Check{}, Tables: []TableRef{},
			Findings: nonNilFindings(e.Findings),
		}
		for _, m := range e.Middleware {
			ep.Middleware = append(ep.Middleware, Middleware{Name: m.Name, Args: m.Args})
		}
		for _, c := range e.Checks {
			ep.Checks = append(ep.Checks, Check{
				Func: c.Func, Enforcement: c.Enforcement,
				Enforced: c.Enforcement == "enforced", File: c.File, Line: c.Line,
			})
		}
		for _, res := range e.Resources {
			ep.Tables = append(ep.Tables, TableRef{Table: res.Table, Operations: nonNil(res.Operations)})
			if tableEndpoints[res.Table] == nil {
				tableEndpoints[res.Table] = map[string]bool{}
				tableOps[res.Table] = map[string]bool{}
			}
			tableEndpoints[res.Table][e.Subject] = true
			for _, op := range res.Operations {
				tableOps[res.Table][op] = true
			}
		}
		s.Endpoints = append(s.Endpoints, ep)
		s.Findings = append(s.Findings, e.Findings...)
		for _, f := range e.Findings {
			s.Counts.Severity[string(f.Severity)]++
		}
	}

	for _, name := range r.Tables {
		s.Tables = append(s.Tables, TableInfo{
			Name:       name,
			Endpoints:  sortedSet(tableEndpoints[name]),
			Operations: sortedSet(tableOps[name]),
		})
	}
	return s
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func nonNilFindings(f []model.Finding) []model.Finding {
	if f == nil {
		return []model.Finding{}
	}
	return f
}

func sortedSet(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
