package schema

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ReadLive reads the authoritative schema from a live PostgreSQL database over
// dbURL, restricted to the public schema. It is the schema source the design
// centers on: information_schema already holds the resolved, current structure,
// so there is nothing to reconstruct and nothing to infer.
//
// The connection URL is used and discarded here; it is never returned, logged
// or stored, so database credentials stay out of scan history.
func ReadLive(ctx context.Context, dbURL string) (*Schema, error) {
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	defer conn.Close(ctx)

	s := &Schema{}
	byName := map[string]*Table{}

	// Tables first, so columns and constraints attach to a known set.
	tableRows, err := conn.Query(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		ORDER BY table_name`)
	if err != nil {
		return nil, fmt.Errorf("read tables: %w", err)
	}
	names, err := collect(tableRows, func(r pgx.Row) (string, error) {
		var n string
		return n, r.Scan(&n)
	})
	if err != nil {
		return nil, fmt.Errorf("read tables: %w", err)
	}
	for _, n := range names {
		s.Tables = append(s.Tables, Table{Name: n})
	}
	// Index the tables only after the slice has stopped growing. Taking the
	// address of an element while still appending would hand out pointers into
	// a backing array that the next append reallocates, silently dropping every
	// column attached through an earlier pointer.
	for i := range s.Tables {
		byName[s.Tables[i].Name] = &s.Tables[i]
	}

	// Columns, in declaration order so the stored shape matches the source.
	colRows, err := conn.Query(ctx, `
		SELECT table_name, column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema = 'public'
		ORDER BY table_name, ordinal_position`)
	if err != nil {
		return nil, fmt.Errorf("read columns: %w", err)
	}
	type colRow struct {
		table, name, typ, nullable string
	}
	cols, err := collect(colRows, func(r pgx.Row) (colRow, error) {
		var c colRow
		return c, r.Scan(&c.table, &c.name, &c.typ, &c.nullable)
	})
	if err != nil {
		return nil, fmt.Errorf("read columns: %w", err)
	}
	for _, c := range cols {
		if t := byName[c.table]; t != nil {
			t.Columns = append(t.Columns, Column{
				Name:     c.name,
				Type:     c.typ,
				Nullable: c.nullable == "YES",
			})
		}
	}

	// Primary keys, in key order.
	if err := readKeyColumns(ctx, conn, "PRIMARY KEY", func(table, col string) {
		if t := byName[table]; t != nil {
			t.PrimaryKey = append(t.PrimaryKey, col)
		}
	}); err != nil {
		return nil, err
	}

	// Unique constraints, grouped by constraint so a multi-column unique key
	// stays one set.
	uniqueByConstraint := map[string]*uniqueGroup{}
	var uniqueOrder []string
	uq, err := conn.Query(ctx, `
		SELECT tc.constraint_name, kcu.table_name, kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		 AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'UNIQUE' AND tc.table_schema = 'public'
		ORDER BY tc.constraint_name, kcu.ordinal_position`)
	if err != nil {
		return nil, fmt.Errorf("read unique constraints: %w", err)
	}
	type uqRow struct{ constraint, table, col string }
	uqs, err := collect(uq, func(r pgx.Row) (uqRow, error) {
		var u uqRow
		return u, r.Scan(&u.constraint, &u.table, &u.col)
	})
	if err != nil {
		return nil, fmt.Errorf("read unique constraints: %w", err)
	}
	for _, u := range uqs {
		g, ok := uniqueByConstraint[u.constraint]
		if !ok {
			g = &uniqueGroup{table: u.table}
			uniqueByConstraint[u.constraint] = g
			uniqueOrder = append(uniqueOrder, u.constraint)
		}
		g.cols = append(g.cols, u.col)
	}
	for _, name := range uniqueOrder {
		g := uniqueByConstraint[name]
		if t := byName[g.table]; t != nil {
			t.Unique = append(t.Unique, g.cols)
		}
	}

	// Explicit foreign keys only, one row per referencing column paired with the
	// column it references. The pairing is read from pg_catalog by position:
	// conkey[i] references confkey[i]. The information_schema join route
	// cross-products a composite key's columns (a 2+2 key yields 4 bogus pairs,
	// three of them relationships that do not exist) — and inventing a
	// relationship the schema never declared is the one thing this reader must
	// never do.
	fkRows, err := conn.Query(ctx, `
		SELECT con.conname,
		       cl.relname  AS from_table,
		       att.attname  AS from_column,
		       clf.relname AS to_table,
		       attf.attname AS to_column
		FROM pg_catalog.pg_constraint con
		JOIN pg_catalog.pg_class cl ON cl.oid = con.conrelid
		JOIN pg_catalog.pg_namespace ns ON ns.oid = cl.relnamespace
		JOIN pg_catalog.pg_class clf ON clf.oid = con.confrelid
		JOIN unnest(con.conkey)  WITH ORDINALITY AS fk(attnum, ord) ON true
		JOIN unnest(con.confkey) WITH ORDINALITY AS pk(attnum, ord) ON pk.ord = fk.ord
		JOIN pg_catalog.pg_attribute att  ON att.attrelid = con.conrelid  AND att.attnum = fk.attnum
		JOIN pg_catalog.pg_attribute attf ON attf.attrelid = con.confrelid AND attf.attnum = pk.attnum
		WHERE con.contype = 'f' AND ns.nspname = 'public'
		ORDER BY from_table, fk.ord`)
	if err != nil {
		return nil, fmt.Errorf("read foreign keys: %w", err)
	}
	fks, err := collect(fkRows, func(r pgx.Row) (ForeignKey, error) {
		var fk ForeignKey
		return fk, r.Scan(&fk.Name, &fk.FromTable, &fk.FromColumn, &fk.ToTable, &fk.ToColumn)
	})
	if err != nil {
		return nil, fmt.Errorf("read foreign keys: %w", err)
	}
	s.ForeignKeys = fks

	return s, nil
}

type uniqueGroup struct {
	table string
	cols  []string
}

// readKeyColumns runs the key-column query for a constraint type and calls emit
// for each (table, column) in key order.
func readKeyColumns(ctx context.Context, conn *pgx.Conn, constraintType string, emit func(table, col string)) error {
	rows, err := conn.Query(ctx, `
		SELECT kcu.table_name, kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		 AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = $1 AND tc.table_schema = 'public'
		ORDER BY kcu.table_name, kcu.ordinal_position`, constraintType)
	if err != nil {
		return fmt.Errorf("read %s columns: %w", constraintType, err)
	}
	defer rows.Close()
	for rows.Next() {
		var table, col string
		if err := rows.Scan(&table, &col); err != nil {
			return fmt.Errorf("read %s columns: %w", constraintType, err)
		}
		emit(table, col)
	}
	return rows.Err()
}

// collect drains rows, mapping each with scan, and returns the slice. It closes
// rows and surfaces any iteration error, so callers do not repeat that dance.
func collect[T any](rows pgx.Rows, scan func(pgx.Row) (T, error)) ([]T, error) {
	defer rows.Close()
	var out []T
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
