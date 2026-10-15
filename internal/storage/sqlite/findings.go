package sqlite

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/YarKhan02/PathWise/internal/model"
)

// InsertFindings writes findings for a scan in one transaction. Like facts,
// findings belong to an immutable scan: a re-run records into a new scan rather
// than editing an old one, so the finding lifecycle (introduced, resolved) can
// be reconstructed by comparing scans. A finding with no ID is assigned a
// sortable one here.
func (s *Store) InsertFindings(ctx context.Context, findings []model.Finding) (int, error) {
	if len(findings) == 0 {
		return 0, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	ins, err := tx.PrepareContext(ctx,
		`INSERT INTO findings
		   (id, scan_id, kind, severity, confidence, endpoint, stable_key,
		    title, detail, remedy, locations, premises, runtime_divergence)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer ins.Close()

	written := 0
	for _, f := range findings {
		if f.ID == "" {
			f.ID = newID("fnd")
		}
		locations, err := json.Marshal(nonNilLocations(f.Locations))
		if err != nil {
			return written, fmt.Errorf("marshal locations for %s: %w", f.StableKey, err)
		}
		premises, err := json.Marshal(nonNilStrings(f.Premises))
		if err != nil {
			return written, fmt.Errorf("marshal premises for %s: %w", f.StableKey, err)
		}

		if _, err := ins.ExecContext(ctx,
			f.ID, f.ScanID, string(f.Kind), string(f.Severity), string(f.Confidence),
			f.Endpoint, f.StableKey, f.Title, f.Detail, f.Remedy,
			string(locations), string(premises), boolToInt(f.RuntimeDivergence),
		); err != nil {
			return written, fmt.Errorf("insert finding %s: %w", f.StableKey, err)
		}
		written++
	}

	if err := tx.Commit(); err != nil {
		return written, err
	}
	return written, nil
}

// Findings reads back the findings recorded for a scan, newest-ordered by id.
func (s *Store) Findings(ctx context.Context, scanID string) ([]model.Finding, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, scan_id, kind, severity, confidence, endpoint, stable_key,
		        title, detail, remedy, locations, premises, runtime_divergence
		 FROM findings WHERE scan_id = ? ORDER BY id`, scanID)
	if err != nil {
		return nil, fmt.Errorf("query findings: %w", err)
	}
	defer rows.Close()

	var out []model.Finding
	for rows.Next() {
		var (
			f              model.Finding
			locations      string
			premises       string
			runtimeDiverge int
		)
		if err := rows.Scan(&f.ID, &f.ScanID, &f.Kind, &f.Severity, &f.Confidence,
			&f.Endpoint, &f.StableKey, &f.Title, &f.Detail, &f.Remedy,
			&locations, &premises, &runtimeDiverge); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(locations), &f.Locations)
		_ = json.Unmarshal([]byte(premises), &f.Premises)
		f.RuntimeDivergence = runtimeDiverge != 0
		out = append(out, f)
	}
	return out, rows.Err()
}

func nonNilLocations(l []model.SourceLocation) []model.SourceLocation {
	if l == nil {
		return []model.SourceLocation{}
	}
	return l
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
