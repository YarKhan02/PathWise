package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YarKhan02/PathWise/internal/report"
	"github.com/YarKhan02/PathWise/internal/server"
)

func TestHandlerServesScanAndPage(t *testing.T) {
	rep := &report.Report{
		Project: "demo", ScanID: "s1", Roles: []string{"admin"}, Tables: []string{"course"},
		RouteCount: 2, RoleCount: 1, TableCount: 1,
		Endpoints: []report.Endpoint{
			{
				Method: "GET", Path: "/course", Subject: "GET /course", Roles: []string{"admin"},
				Resources: []report.Resource{{Table: "course", Operations: []string{"SELECT"}}},
			},
			// No gate, no tables: its slices must serialize as [], never null.
			{Method: "POST", Path: "/login", Subject: "POST /login"},
		},
	}
	h := server.Handler(server.BuildScan(rep))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/scan", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/api/scan status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "null") {
		t.Errorf("scan JSON contains null (nil slice leaked): %s", rec.Body.String())
	}
	var s server.Scan
	if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
		t.Fatalf("decode scan: %v", err)
	}
	if len(s.Endpoints) != 2 || s.Endpoints[0].Path != "/course" {
		t.Errorf("endpoints = %+v", s.Endpoints)
	}
	if len(s.Tables) != 1 || s.Tables[0].Name != "course" || len(s.Tables[0].Endpoints) != 1 {
		t.Errorf("tables = %+v", s.Tables)
	}

	// Client-side route with no file falls back to the SPA entrypoint.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/endpoints", nil))
	if rec.Code != http.StatusOK && rec.Code != http.StatusNotImplemented {
		t.Errorf("SPA fallback status = %d (want 200 with assets, or 501 before they are built)", rec.Code)
	}
}
