package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/YarKhan02/PathWise/internal/model"
)

// SchemaSource records where schema facts came from. A finding resting on an
// inferred schema is weaker than one resting on the live database, and the
// reader has to be able to tell them apart.
type SchemaSource string

const (
	// SchemaLive means schema was read from the running PostgreSQL instance.
	SchemaLive SchemaSource = "live"

	// SchemaMigrationsOnly is the degraded mode: no database was reachable,
	// so schema was reconstructed by replaying migrations. Schema facts in
	// this mode are inferred, not code_confirmed.
	SchemaMigrationsOnly SchemaSource = "migrations_only"

	// SchemaNone means no schema was available at all.
	SchemaNone SchemaSource = "none"
)

// ScanStatus is a scan's lifecycle state.
type ScanStatus string

const (
	ScanRunning  ScanStatus = "running"
	ScanComplete ScanStatus = "complete"
	ScanFailed   ScanStatus = "failed"
)

// newID returns a lexicographically sortable identifier: a millisecond
// timestamp prefix plus random suffix. Sortable IDs mean "the latest scan" is
// an ORDER BY rather than a join against timestamps.
func newID(prefix string) string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%s_%013d_%s", prefix, time.Now().UTC().UnixMilli(), hex.EncodeToString(b[:]))
}

func nowRFC3339() string { return time.Now().UTC().Format(time.RFC3339Nano) }

// Project is a stored project row.
type Project struct {
	ID         string
	Name       string
	RootPath   string
	ModulePath string
}

// UpsertProject returns the project for rootPath, creating it if absent.
// Projects are keyed by root path so that repeated scans of one checkout
// accumulate history rather than forking into separate projects.
func (s *Store) UpsertProject(ctx context.Context, name, rootPath, modulePath string) (*Project, error) {
	var p Project
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, root_path, module_path FROM projects WHERE root_path = ?`,
		rootPath,
	).Scan(&p.ID, &p.Name, &p.RootPath, &p.ModulePath)

	switch {
	case err == nil:
		if p.ModulePath != modulePath || p.Name != name {
			if _, err := s.db.ExecContext(ctx,
				`UPDATE projects SET module_path = ?, name = ? WHERE id = ?`,
				modulePath, name, p.ID); err != nil {
				return nil, fmt.Errorf("update project: %w", err)
			}
			p.ModulePath, p.Name = modulePath, name
		}
		return &p, nil

	case errors.Is(err, sql.ErrNoRows):
		p = Project{ID: newID("prj"), Name: name, RootPath: rootPath, ModulePath: modulePath}
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO projects (id, name, root_path, module_path, created_at)
			 VALUES (?, ?, ?, ?, ?)`,
			p.ID, p.Name, p.RootPath, p.ModulePath, nowRFC3339()); err != nil {
			return nil, fmt.Errorf("insert project: %w", err)
		}
		return &p, nil

	default:
		return nil, fmt.Errorf("lookup project: %w", err)
	}
}

// Scan is a stored scan row.
type Scan struct {
	ID              string
	ProjectID       string
	GitRevision     string
	GitDirty        bool
	AnalyzerVersion string
	SchemaSource    SchemaSource
	DisabledChecks  []model.FindingKind
	Status          ScanStatus
	Error           string
	StartedAt       string
	FinishedAt      string
}

// BeginScan creates an immutable scan record in the running state.
func (s *Store) BeginScan(ctx context.Context, projectID, gitRev string, dirty bool,
	analyzerVersion string, schemaSource SchemaSource, disabled []model.FindingKind) (*Scan, error) {

	if disabled == nil {
		disabled = []model.FindingKind{}
	}
	encoded, err := json.Marshal(disabled)
	if err != nil {
		return nil, err
	}

	sc := &Scan{
		ID:              newID("scan"),
		ProjectID:       projectID,
		GitRevision:     gitRev,
		GitDirty:        dirty,
		AnalyzerVersion: analyzerVersion,
		SchemaSource:    schemaSource,
		DisabledChecks:  disabled,
		Status:          ScanRunning,
		StartedAt:       nowRFC3339(),
	}

	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO scans (id, project_id, git_revision, git_dirty, analyzer_version,
		                    schema_source, disabled_checks, status, started_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sc.ID, sc.ProjectID, sc.GitRevision, boolToInt(sc.GitDirty), sc.AnalyzerVersion,
		string(sc.SchemaSource), string(encoded), string(sc.Status), sc.StartedAt,
	); err != nil {
		return nil, fmt.Errorf("insert scan: %w", err)
	}
	return sc, nil
}

// FinishScan marks a scan complete or failed. Scans are never deleted or
// rewritten beyond this terminal transition.
func (s *Store) FinishScan(ctx context.Context, scanID string, scanErr error) error {
	status, msg := ScanComplete, ""
	if scanErr != nil {
		status, msg = ScanFailed, scanErr.Error()
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE scans SET status = ?, error = ?, finished_at = ? WHERE id = ?`,
		string(status), msg, nowRFC3339(), scanID); err != nil {
		return fmt.Errorf("finish scan: %w", err)
	}
	return nil
}

// LatestScan returns the most recent scan for a project, or nil if none.
func (s *Store) LatestScan(ctx context.Context, projectID string) (*Scan, error) {
	var (
		sc       Scan
		dirty    int
		disabled string
		finished sql.NullString
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, git_revision, git_dirty, analyzer_version,
		        schema_source, disabled_checks, status, error, started_at, finished_at
		 FROM scans WHERE project_id = ? ORDER BY id DESC LIMIT 1`,
		projectID,
	).Scan(&sc.ID, &sc.ProjectID, &sc.GitRevision, &dirty, &sc.AnalyzerVersion,
		&sc.SchemaSource, &disabled, &sc.Status, &sc.Error, &sc.StartedAt, &finished)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("latest scan: %w", err)
	}

	sc.GitDirty = dirty != 0
	sc.FinishedAt = finished.String
	_ = json.Unmarshal([]byte(disabled), &sc.DisabledChecks)
	return &sc, nil
}

// InsertFacts writes facts in one transaction, skipping duplicates within the
// scan. Facts are append-only: an analyzer may be run again, but it writes
// into a new scan rather than editing an old one.
func (s *Store) InsertFacts(ctx context.Context, facts []model.Fact) (int, error) {
	if len(facts) == 0 {
		return 0, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	insFact, err := tx.PrepareContext(ctx,
		`INSERT OR IGNORE INTO facts
		   (scan_id, kind, subject, attrs, status, unresolved_reason, details, scenario_id, fingerprint)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer insFact.Close()

	insLoc, err := tx.PrepareContext(ctx,
		`INSERT INTO fact_locations (fact_id, file, start_line, end_line) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer insLoc.Close()

	written := 0
	for _, f := range facts {
		if !f.Evidence.Valid() {
			return written, fmt.Errorf(
				"refusing to store malformed evidence for %s %q: status %q is missing its required field",
				f.Kind, f.Subject, f.Evidence.Status)
		}

		attrs := []byte("{}")
		if len(f.Attrs) > 0 {
			if attrs, err = json.Marshal(f.Attrs); err != nil {
				return written, fmt.Errorf("marshal attrs for %s: %w", f.Subject, err)
			}
		}

		res, err := insFact.ExecContext(ctx,
			f.ScanID, string(f.Kind), f.Subject, string(attrs),
			string(f.Evidence.Status), string(f.Evidence.UnresolvedReason),
			f.Evidence.Details, f.Evidence.ScenarioID, f.Fingerprint())
		if err != nil {
			return written, fmt.Errorf("insert fact %s: %w", f.Subject, err)
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return written, err
		}
		if affected == 0 {
			continue // duplicate fingerprint within this scan
		}
		written++

		id, err := res.LastInsertId()
		if err != nil {
			return written, err
		}
		for _, loc := range f.Evidence.Source {
			if _, err := insLoc.ExecContext(ctx, id, loc.File, loc.StartLine, loc.EndLine); err != nil {
				return written, fmt.Errorf("insert location for %s: %w", f.Subject, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return written, err
	}
	return written, nil
}

// FactsByKind reads back the facts of one kind for a scan, locations included.
func (s *Store) FactsByKind(ctx context.Context, scanID string, kind model.FactKind) ([]model.Fact, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, scan_id, kind, subject, attrs, status, unresolved_reason, details, scenario_id
		 FROM facts WHERE scan_id = ? AND kind = ? ORDER BY id`,
		scanID, string(kind))
	if err != nil {
		return nil, fmt.Errorf("query facts: %w", err)
	}
	defer rows.Close()

	var out []model.Fact
	for rows.Next() {
		var (
			f     model.Fact
			attrs string
		)
		if err := rows.Scan(&f.ID, &f.ScanID, &f.Kind, &f.Subject, &attrs,
			&f.Evidence.Status, &f.Evidence.UnresolvedReason,
			&f.Evidence.Details, &f.Evidence.ScenarioID); err != nil {
			return nil, err
		}
		if attrs != "" && attrs != "{}" {
			_ = json.Unmarshal([]byte(attrs), &f.Attrs)
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range out {
		locs, err := s.locationsFor(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Evidence.Source = locs
	}
	return out, nil
}

func (s *Store) locationsFor(ctx context.Context, factID int64) ([]model.SourceLocation, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT file, start_line, end_line FROM fact_locations WHERE fact_id = ?`, factID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.SourceLocation
	for rows.Next() {
		var l model.SourceLocation
		if err := rows.Scan(&l.File, &l.StartLine, &l.EndLine); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// CountFacts returns the number of facts recorded for a scan.
func (s *Store) CountFacts(ctx context.Context, scanID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM facts WHERE scan_id = ?`, scanID).Scan(&n)
	return n, err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
