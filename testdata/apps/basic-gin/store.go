package main

import (
	"context"
	"errors"
)

var errUnauthorized = errors.New("unauthorized")

// Student is the row shape returned by the student handlers.
type Student struct {
	ID      int64  `json:"id"`
	OrgID   int64  `json:"org_id"`
	BatchID int64  `json:"batch_id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
}

// StudentProfile carries national_id, which is declared sensitive in
// .pathwise.yaml. Its presence in a JSON response is what the
// response_field_exceeds_scope check looks for.
type StudentProfile struct {
	ID         int64  `json:"id"`
	OrgID      int64  `json:"org_id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	NationalID string `json:"national_id"`
}

// Batch is the row shape returned by ListBatches.
type Batch struct {
	ID    int64  `json:"id"`
	OrgID int64  `json:"org_id"`
	Name  string `json:"name"`
}

// studentByID filters on the primary key alone. The id is caller-supplied and
// no identity-derived predicate constrains the row, which is the query half of
// the IDOR in GetStudent.
func studentByID(ctx context.Context, id string) (*Student, error) {
	const q = `
		SELECT id, org_id, batch_id, name, email
		FROM students
		WHERE id = $1`

	var s Student
	err := db.QueryRowContext(ctx, q, id).
		Scan(&s.ID, &s.OrgID, &s.BatchID, &s.Name, &s.Email)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// studentProfileByID selects national_id alongside the ordinary columns.
func studentProfileByID(ctx context.Context, id string) (*StudentProfile, error) {
	const q = `
		SELECT id, org_id, name, email, national_id
		FROM students
		WHERE id = $1`

	var p StudentProfile
	err := db.QueryRowContext(ctx, q, id).
		Scan(&p.ID, &p.OrgID, &p.Name, &p.Email, &p.NationalID)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// allBatches has no tenant predicate at all.
func allBatches(ctx context.Context) ([]Batch, error) {
	const q = `
		SELECT id, org_id, name
		FROM batches
		ORDER BY name`

	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Batch
	for rows.Next() {
		var b Batch
		if err := rows.Scan(&b.ID, &b.OrgID, &b.Name); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// reportRows is correctly scoped: the tenant predicate is bound to a value
// derived from the authenticated principal, not from the request.
func reportRows(ctx context.Context, orgID int64) ([]Batch, error) {
	const q = `
		SELECT id, org_id, name
		FROM batches
		WHERE org_id = $1`

	rows, err := db.QueryContext(ctx, q, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Batch
	for rows.Next() {
		var b Batch
		if err := rows.Scan(&b.ID, &b.OrgID, &b.Name); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// dashboardRows is tenant-scoped in the same way as reportRows.
func dashboardRows(ctx context.Context, orgID int64) ([]Batch, error) {
	return reportRows(ctx, orgID)
}

// rawStudentQuery executes caller-assembled SQL. The query text is opaque to
// static analysis at this call site.
func rawStudentQuery(ctx context.Context, query string) ([]Student, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Student
	for rows.Next() {
		var s Student
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// lookupSession resolves a session token to a principal.
func lookupSession(token string) (*User, error) {
	const q = `
		SELECT u.id, u.org_id, u.role
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token = $1 AND s.expires_at > now()`

	var u User
	err := db.QueryRow(q, token).Scan(&u.ID, &u.OrgID, &u.Role)
	if err != nil {
		return nil, errUnauthorized
	}
	return &u, nil
}

// createSession issues a session token for valid credentials.
func createSession(ctx context.Context, email, password string) (string, error) {
	const q = `
		SELECT id, org_id, role
		FROM users
		WHERE email = $1 AND password_hash = crypt($2, password_hash)`

	var u User
	if err := db.QueryRowContext(ctx, q, email, password).
		Scan(&u.ID, &u.OrgID, &u.Role); err != nil {
		return "", errUnauthorized
	}

	const ins = `
		INSERT INTO sessions (user_id, token, expires_at)
		VALUES ($1, $2, now() + interval '24 hours')
		RETURNING token`

	var token string
	if err := db.QueryRowContext(ctx, ins, u.ID, newToken()).Scan(&token); err != nil {
		return "", err
	}
	return token, nil
}

func newToken() string { return "generated-token" }
