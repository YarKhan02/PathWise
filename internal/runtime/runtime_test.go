package runtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockApp stands in for a real application. /batches is public; /admin/reports
// demands the admin role but is deliberately broken — it serves everyone,
// which is the divergence the tool must catch. /dashboard is correctly
// admin-only. Roles are read from the bearer token, which the test sets to the
// identity's role name.
func mockApp() http.Handler {
	role := func(r *http.Request) string {
		const p = "Bearer "
		if h := r.Header.Get("Authorization"); len(h) > len(p) {
			return h[len(p):]
		}
		return ""
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/batches", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	// Broken: ignores the caller's role entirely.
	mux.HandleFunc("/admin/reports", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	// Correct: admin only.
	mux.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		if role(r) != "admin" {
			w.WriteHeader(403)
			return
		}
		w.WriteHeader(200)
	})
	return mux
}

func TestRunClassifiesOutcomes(t *testing.T) {
	srv := httptest.NewServer(mockApp())
	defer srv.Close()

	endpoints := []Endpoint{
		{Method: "GET", Path: "/batches"},                                 // public
		{Method: "GET", Path: "/admin/reports", Roles: []string{"admin"}}, // broken gate
		{Method: "GET", Path: "/dashboard", Roles: []string{"admin"}},     // correct gate
		{Method: "GET", Path: "/students/:id"},                            // parameterized → unverified
		{Method: "POST", Path: "/batches"},                                // non-GET → unverified
	}
	identities := []Identity{
		{Role: "admin", Token: "admin"},
		{Role: "student", Token: "student"},
	}

	results := Run(context.Background(), srv.URL, Plan(endpoints, identities, nil), srv.Client())

	// Index by (path, role) for assertions.
	got := map[string]Outcome{}
	for _, r := range results {
		got[r.Method+" "+r.Path+"|"+r.Identity.Role] = r.Outcome
	}

	want := map[string]Outcome{
		// Public: both predicted and observed allowed.
		"GET /batches|admin":   OutcomePass,
		"GET /batches|student": OutcomePass,
		// Admin can reach admin-gated endpoints as predicted.
		"GET /admin/reports|admin": OutcomePass,
		"GET /dashboard|admin":     OutcomePass,
		// The whole point: student should be blocked from /admin/reports, but the
		// broken app serves it → divergent, the strongest finding.
		"GET /admin/reports|student": OutcomeDivergent,
		// /dashboard is correctly protected: student is denied as predicted.
		"GET /dashboard|student": OutcomePass,
		// A path parameter has no known value → analysis gap, unverified.
		"GET /students/:id|admin":   OutcomeUnverified,
		"GET /students/:id|student": OutcomeUnverified,
		// A non-GET method is refused by the safety classifier → blocked (policy),
		// which is distinct from unverified (could not conclude).
		"POST /batches|admin":   OutcomeBlocked,
		"POST /batches|student": OutcomeBlocked,
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s: outcome = %q, want %q", k, got[k], w)
		}
	}
}

// TestFailWhenPredictedReachableButDenied covers the fourth outcome: the
// analysis said an identity should reach an endpoint, the app denied it.
func TestFailWhenPredictedReachableButDenied(t *testing.T) {
	srv := httptest.NewServer(mockApp())
	defer srv.Close()

	// Static analysis found no gate on /dashboard (Roles empty), so it predicts
	// every identity may reach it. The app actually requires admin, so a student
	// is denied — a disagreement worth surfacing, but not a vulnerability.
	endpoints := []Endpoint{{Method: "GET", Path: "/dashboard"}}
	ids := []Identity{{Role: "student", Token: "student"}}

	results := Run(context.Background(), srv.URL, Plan(endpoints, ids, nil), srv.Client())
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	if results[0].Outcome != OutcomeFail {
		t.Errorf("outcome = %q, want %q", results[0].Outcome, OutcomeFail)
	}
}

// TestNonGetOptIn covers the safety classifier's per-endpoint opt-in: a non-GET
// endpoint is blocked by default, but executes when the developer allows it.
func TestNonGetOptIn(t *testing.T) {
	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			hit = true
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	endpoints := []Endpoint{{Method: "DELETE", Path: "/batches/all"}}
	ids := []Identity{{Role: "admin", Token: "admin"}}

	// Default: blocked, never fired.
	blocked := Run(context.Background(), srv.URL, Plan(endpoints, ids, nil), srv.Client())
	if blocked[0].Outcome != OutcomeBlocked {
		t.Errorf("without opt-in: outcome = %q, want %q", blocked[0].Outcome, OutcomeBlocked)
	}
	if hit {
		t.Error("blocked DELETE must not reach the server")
	}

	// Opted in: executed, and because no gate was predicted it passes.
	allow := map[string]bool{Key("DELETE", "/batches/all"): true}
	ran := Run(context.Background(), srv.URL, Plan(endpoints, ids, allow), srv.Client())
	if !hit {
		t.Error("opted-in DELETE should reach the server")
	}
	if ran[0].Outcome != OutcomePass {
		t.Errorf("with opt-in: outcome = %q, want %q", ran[0].Outcome, OutcomePass)
	}
}
