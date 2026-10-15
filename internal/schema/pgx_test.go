package schema_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/YarKhan02/PathWise/internal/corpus"
	"github.com/YarKhan02/PathWise/internal/schema"
)

// TestReadLiveMatchesCorpus applies the basic-gin migration to a real
// PostgreSQL database and checks that the live reader recovers exactly the
// schema expected.yaml declares — the Phase 1 "all tables and FKs recovered"
// exit criterion for the authoritative path.
//
// It needs a Postgres to connect to, named by PATHWISE_TEST_DB (a libpq DSN or
// URL to an EMPTY database the test may create tables in). Without it the test
// skips, so `go test ./...` stays green on a machine with no database while the
// reader is still exercised against a real catalog wherever one is available.
func TestReadLiveMatchesCorpus(t *testing.T) {
	dsn := os.Getenv("PATHWISE_TEST_DB")
	if dsn == "" {
		t.Skip("set PATHWISE_TEST_DB to a libpq DSN for an empty database to run the live schema test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	corpusDir, err := filepath.Abs(filepath.Join("..", "..", "testdata", "apps", "basic-gin"))
	if err != nil {
		t.Fatalf("resolve corpus path: %v", err)
	}

	applyMigration(t, ctx, dsn, filepath.Join(corpusDir, "migrations", "0001_init.up.sql"))

	got, err := schema.ReadLive(ctx, dsn)
	if err != nil {
		t.Fatalf("ReadLive: %v", err)
	}

	app, err := corpus.LoadApp(corpusDir)
	if err != nil {
		t.Fatalf("load corpus expectations: %v", err)
	}
	want := app.Schema

	// Tables and their columns (compared as sets; information_schema order is
	// ordinal, expected.yaml order is declaration — both describe the same set).
	if len(got.Tables) != len(want.Tables) {
		t.Errorf("tables: got %d, want %d", len(got.Tables), len(want.Tables))
	}
	for name, wantTable := range want.Tables {
		gt := got.Find(name)
		if gt == nil {
			t.Errorf("table %q not recovered", name)
			continue
		}
		if !sameSet(columnNames(gt), wantTable.Columns) {
			t.Errorf("table %q columns: got %v, want %v", name, columnNames(gt), wantTable.Columns)
		}
		if !sameSet(gt.PrimaryKey, wantTable.PrimaryKey) {
			t.Errorf("table %q primary key: got %v, want %v", name, gt.PrimaryKey, wantTable.PrimaryKey)
		}
	}

	// Foreign keys, compared as a set of "from -> to" edges.
	gotFK := map[string]bool{}
	for _, fk := range got.ForeignKeys {
		gotFK[fk.FromTable+"."+fk.FromColumn+" -> "+fk.ToTable+"."+fk.ToColumn] = true
	}
	for _, wfk := range want.ForeignKeys {
		edge := wfk.From + " -> " + wfk.To
		if !gotFK[edge] {
			t.Errorf("foreign key %q not recovered (got %v)", edge, keys(gotFK))
		}
	}
	if len(got.ForeignKeys) != len(want.ForeignKeys) {
		t.Errorf("foreign keys: got %d, want %d", len(got.ForeignKeys), len(want.ForeignKeys))
	}
}

// TestReadLiveCompositeForeignKey guards the composite-key pairing. A
// multi-column foreign key must be read as its real column pairs, not as the
// cross-product of referencing and referenced columns — inventing an edge the
// schema never declared is exactly what the reader must never do.
func TestReadLiveCompositeForeignKey(t *testing.T) {
	dsn := os.Getenv("PATHWISE_TEST_DB")
	if dsn == "" {
		t.Skip("set PATHWISE_TEST_DB to a libpq DSN for an empty database to run the live schema test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resetAndExec(t, ctx, dsn, `
		CREATE TABLE parent (a int, b int, PRIMARY KEY (a, b));
		CREATE TABLE child  (a int, b int, FOREIGN KEY (a, b) REFERENCES parent (a, b));`)

	got, err := schema.ReadLive(ctx, dsn)
	if err != nil {
		t.Fatalf("ReadLive: %v", err)
	}

	// The only correct reading is two edges: a->a and b->b. The cross-product
	// bug would add a->b and b->a.
	want := map[string]bool{"child.a -> parent.a": true, "child.b -> parent.b": true}
	for _, fk := range got.ForeignKeys {
		edge := fk.FromTable + "." + fk.FromColumn + " -> " + fk.ToTable + "." + fk.ToColumn
		if !want[edge] {
			t.Errorf("unexpected foreign key %q (cross-product bug?)", edge)
		}
	}
	if len(got.ForeignKeys) != len(want) {
		t.Errorf("got %d foreign keys, want %d", len(got.ForeignKeys), len(want))
	}
}

// applyMigration resets the public schema and runs one SQL file.
func applyMigration(t *testing.T, ctx context.Context, dsn, file string) {
	t.Helper()
	sqlBytes, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	resetAndExec(t, ctx, dsn, string(sqlBytes))
}

// resetAndExec drops and recreates the public schema, then runs stmts, so each
// schema test starts from a known-empty database and is repeatable.
func resetAndExec(t *testing.T, ctx context.Context, dsn, stmts string) {
	t.Helper()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if _, err := conn.Exec(ctx, stmts); err != nil {
		t.Fatalf("exec schema: %v", err)
	}
}

func columnNames(t *schema.Table) []string {
	out := make([]string, len(t.Columns))
	for i, c := range t.Columns {
		out[i] = c.Name
	}
	return out
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	as := append([]string(nil), a...)
	bs := append([]string(nil), b...)
	sort.Strings(as)
	sort.Strings(bs)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
