// Package corpus loads the test corpus: small Go applications whose
// authorization structure is known in advance, paired with a declaration of
// what PathWise is expected to find in them.
//
// This exists because without known-answer applications there is no way to
// distinguish "found nothing" from "found nothing because the analyzer
// silently gave up on an interface dispatch". That ambiguity would quietly
// invalidate every later milestone, so the corpus is built before the
// analyzers that it judges.
//
// Expectations are anchored to marker comments in the corpus source rather
// than to line numbers, so that editing a corpus app cannot silently
// invalidate its expectations:
//
//	// pathwise:expect kind=missing_ownership_check id=idor-get-student
//	// pathwise:safe id=ownership-enforced
//	// pathwise:unresolved reason=dynamic_sql id=dynamic-sql-search
package corpus

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/YarKhan02/pathwise/internal/model"
)

// ExpectationFile is the name of the per-app expectation declaration.
const ExpectationFile = "expected.yaml"

// App is one corpus application and its known answers.
type App struct {
	// Dir is the app's directory, filled in on load.
	Dir string `yaml:"-"`

	Name        string `yaml:"app"`
	Framework   string `yaml:"framework"`
	QueryLayer  string `yaml:"query_layer"`
	Description string `yaml:"description"`

	Routes []ExpectedRoute `yaml:"routes"`
	Schema ExpectedSchema  `yaml:"schema"`

	// Findings are the flaws deliberately seeded in this app. Every one must
	// be detected for the app to pass.
	Findings []ExpectedFinding `yaml:"expected_findings"`

	// Unresolved are sites analysis is expected to give up on, naming why.
	// These are negative controls: reporting a finding here is a false
	// positive, and reporting nothing at all is a silent failure.
	Unresolved []ExpectedUnresolved `yaml:"expected_unresolved"`

	// Safe are correctly-authorized sites. A finding on any of these is a
	// false positive.
	Safe []ExpectedSafe `yaml:"expected_safe"`
}

// ExpectedRoute is a route the discovery stage must recover.
type ExpectedRoute struct {
	Method     string   `yaml:"method"`
	Pattern    string   `yaml:"pattern"`
	Handler    string   `yaml:"handler"`
	Middleware []string `yaml:"middleware"`
}

// Subject is the route's normalized stable identity.
func (r ExpectedRoute) Subject() string {
	return model.RouteSubject(r.Method, r.Pattern)
}

// ExpectedSchema is the schema the schema stage must recover.
type ExpectedSchema struct {
	Tables      map[string]ExpectedTable `yaml:"tables"`
	ForeignKeys []ExpectedForeignKey     `yaml:"foreign_keys"`
}

// ExpectedTable is one table's expected shape.
type ExpectedTable struct {
	Columns    []string `yaml:"columns"`
	PrimaryKey []string `yaml:"primary_key"`
}

// ExpectedForeignKey is an explicit foreign-key relationship.
type ExpectedForeignKey struct {
	From string `yaml:"from"` // table.column
	To   string `yaml:"to"`   // table.column
}

// ExpectedFinding is a seeded flaw.
type ExpectedFinding struct {
	// ID matches a pathwise:expect marker in the corpus source.
	ID string `yaml:"id"`

	Kind     model.FindingKind `yaml:"kind"`
	Endpoint string            `yaml:"endpoint"`
	Function string            `yaml:"function"`

	// Why states the reasoning a correct analyzer should follow. It is
	// documentation for whoever debugs a miss, not an assertion.
	Why string `yaml:"why"`

	// RequiresConfig notes that this finding only fires with config present.
	RequiresConfig bool `yaml:"requires_config"`
}

// ExpectedUnresolved is a site analysis must report as unverified.
type ExpectedUnresolved struct {
	ID       string                 `yaml:"id"`
	Reason   model.UnresolvedReason `yaml:"reason"`
	Function string                 `yaml:"function"`
	Why      string                 `yaml:"why"`
}

// ExpectedSafe is a correctly-authorized site.
type ExpectedSafe struct {
	ID       string `yaml:"id"`
	Function string `yaml:"function"`
	Why      string `yaml:"why"`
}

// LoadApp reads one corpus app's expectations.
func LoadApp(dir string) (*App, error) {
	path := filepath.Join(dir, ExpectationFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var app App
	if err := yaml.Unmarshal(raw, &app); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	app.Dir = dir
	return &app, nil
}

// LoadAll reads every app under root that has an expectation file.
func LoadAll(root string) ([]*App, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read corpus root %s: %w", root, err)
	}

	var apps []*App
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		if _, err := os.Stat(filepath.Join(dir, ExpectationFile)); err != nil {
			continue
		}
		app, err := LoadApp(dir)
		if err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}

	sort.Slice(apps, func(i, j int) bool { return apps[i].Name < apps[j].Name })
	return apps, nil
}

// Validate cross-checks the expectation file against the markers in the
// corpus source. It catches the two ways a corpus silently decays: an
// expectation whose marker was deleted, and a marker nobody declared.
func (a *App) Validate() []error {
	markers, err := ScanMarkers(a.Dir)
	if err != nil {
		return []error{err}
	}

	declared := map[string]string{} // id -> kind of declaration
	var errs []error

	add := func(id, what string) {
		if id == "" {
			errs = append(errs, fmt.Errorf("%s: %s declaration has an empty id", a.Name, what))
			return
		}
		if prev, dup := declared[id]; dup {
			errs = append(errs, fmt.Errorf("%s: id %q declared twice (%s and %s)", a.Name, id, prev, what))
			return
		}
		declared[id] = what
	}

	for _, f := range a.Findings {
		add(f.ID, "expected_findings")
		if f.Kind == "" {
			errs = append(errs, fmt.Errorf("%s: finding %q has no kind", a.Name, f.ID))
		}
		if f.RequiresConfig != f.Kind.RequiresConfig() {
			errs = append(errs, fmt.Errorf(
				"%s: finding %q declares requires_config=%v but kind %s requires %v",
				a.Name, f.ID, f.RequiresConfig, f.Kind, f.Kind.RequiresConfig()))
		}
	}
	for _, u := range a.Unresolved {
		add(u.ID, "expected_unresolved")
		if u.Reason == model.ReasonNone {
			errs = append(errs, fmt.Errorf("%s: unresolved %q has no reason", a.Name, u.ID))
		}
	}
	for _, s := range a.Safe {
		add(s.ID, "expected_safe")
	}

	markerIDs := map[string]Marker{}
	for _, m := range markers {
		if m.ID == "" {
			errs = append(errs, fmt.Errorf("%s: marker at %s:%d has no id", a.Name, m.File, m.Line))
			continue
		}
		if prev, dup := markerIDs[m.ID]; dup {
			errs = append(errs, fmt.Errorf("%s: marker id %q appears at %s:%d and %s:%d",
				a.Name, m.ID, prev.File, prev.Line, m.File, m.Line))
			continue
		}
		markerIDs[m.ID] = m
	}

	for id, what := range declared {
		if _, ok := markerIDs[id]; !ok {
			errs = append(errs, fmt.Errorf(
				"%s: %s declares id %q but no marker comment in the source carries it",
				a.Name, what, id))
		}
	}
	for id, m := range markerIDs {
		if _, ok := declared[id]; !ok {
			errs = append(errs, fmt.Errorf(
				"%s: marker %q at %s:%d is not declared in %s",
				a.Name, id, m.File, m.Line, ExpectationFile))
		}
	}

	for _, f := range a.Findings {
		if f.Endpoint == "" {
			continue
		}
		if !a.hasRouteSubject(f.Endpoint) {
			errs = append(errs, fmt.Errorf(
				"%s: finding %q names endpoint %q which is not in the declared routes",
				a.Name, f.ID, f.Endpoint))
		}
	}

	for _, fk := range a.Schema.ForeignKeys {
		for _, ref := range []string{fk.From, fk.To} {
			table, col, ok := strings.Cut(ref, ".")
			if !ok {
				errs = append(errs, fmt.Errorf("%s: foreign key %q is not table.column", a.Name, ref))
				continue
			}
			t, exists := a.Schema.Tables[table]
			if !exists {
				errs = append(errs, fmt.Errorf("%s: foreign key references unknown table %q", a.Name, table))
				continue
			}
			if !contains(t.Columns, col) {
				errs = append(errs, fmt.Errorf(
					"%s: foreign key references unknown column %q on table %q", a.Name, col, table))
			}
		}
	}

	return errs
}

func (a *App) hasRouteSubject(subject string) bool {
	want := subject
	if parts := strings.Fields(subject); len(parts) == 2 {
		want = model.RouteSubject(parts[0], parts[1])
	}
	for _, r := range a.Routes {
		if r.Subject() == want {
			return true
		}
	}
	return false
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
