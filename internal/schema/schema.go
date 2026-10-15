// Package schema reads the database schema PathWise analyzes against: the
// tables, columns, primary keys, unique constraints and explicit foreign keys
// the authorization checks are reasoned over.
//
// Two sources, deliberately unequal. The live PostgreSQL catalog is
// authoritative — it is the schema the running application actually sees, so a
// finding resting on it is as strong as the schema gets. Migrations are a
// degraded fallback for when no database is reachable; a schema reconstructed
// by replaying DDL can be wrong (a migration the project never ran, a manual
// change made outside migrations), so facts from that path are marked inferred
// and the scan's schema_source records which path produced them.
//
// One rule spans both sources: only relationships the schema actually declares
// are recorded. PathWise never invents a foreign key because two columns share
// a name — an un-constrained org_id is not evidence of a relationship, and
// treating it as one would manufacture findings.
package schema

import (
	"github.com/YarKhan02/pathwise/internal/model"
)

// Schema is a database's structure as PathWise needs it. It is intentionally
// smaller than the real catalog: indexes, defaults and check constraints do not
// bear on reachability of rows or fields, so they are not carried.
type Schema struct {
	Tables      []Table
	ForeignKeys []ForeignKey
}

// Table is one base table.
type Table struct {
	Name       string
	Columns    []Column
	PrimaryKey []string   // column names, in key order
	Unique     [][]string // each unique constraint as its set of columns
}

// Column is one column's shape. Nullability matters because a nullable
// tenant-discriminator column changes what "has a tenant predicate" means.
type Column struct {
	Name     string
	Type     string
	Nullable bool
}

// ForeignKey is one explicit foreign-key relationship, a single referencing
// column paired with the single column it references. A composite key is
// recorded as several ForeignKeys that share a Name, one per column pair in
// declaration order — each pair is a real relationship the schema declares, not
// a cross-product. The reader relies on pg_catalog to keep the pairing correct;
// see ReadLive.
type ForeignKey struct {
	Name       string
	FromTable  string
	FromColumn string
	ToTable    string
	ToColumn   string
}

// Find returns the named table, or nil.
func (s *Schema) Find(name string) *Table {
	for i := range s.Tables {
		if s.Tables[i].Name == name {
			return &s.Tables[i]
		}
	}
	return nil
}

// Facts projects the schema into the shared model, tied to one scan. The
// evidence carries the source distinction: live schema is code_confirmed from
// the catalog, a migration-reconstructed schema is inferred. Callers pass the
// evidence so this projection stays identical across both sources.
func (s *Schema) Facts(scanID string, ev model.Evidence) []model.Fact {
	var facts []model.Fact

	for _, t := range s.Tables {
		cols := make([]string, len(t.Columns))
		for i, c := range t.Columns {
			cols[i] = c.Name
		}
		attrs := map[string]any{"columns": cols}
		if len(t.PrimaryKey) > 0 {
			attrs["primary_key"] = t.PrimaryKey
		}
		if len(t.Unique) > 0 {
			attrs["unique"] = t.Unique
		}
		facts = append(facts, model.Fact{
			ScanID:   scanID,
			Kind:     model.FactTable,
			Subject:  t.Name,
			Attrs:    attrs,
			Evidence: ev,
		})

		for _, c := range t.Columns {
			facts = append(facts, model.Fact{
				ScanID:  scanID,
				Kind:    model.FactColumn,
				Subject: t.Name + "." + c.Name,
				Attrs: map[string]any{
					"table":    t.Name,
					"type":     c.Type,
					"nullable": c.Nullable,
				},
				Evidence: ev,
			})
		}

		if len(t.PrimaryKey) > 0 {
			facts = append(facts, model.Fact{
				ScanID:   scanID,
				Kind:     model.FactPrimaryKey,
				Subject:  t.Name,
				Attrs:    map[string]any{"columns": t.PrimaryKey},
				Evidence: ev,
			})
		}
	}

	for _, fk := range s.ForeignKeys {
		facts = append(facts, model.Fact{
			ScanID:  scanID,
			Kind:    model.FactForeignKey,
			Subject: fk.FromTable + "." + fk.FromColumn + " -> " + fk.ToTable + "." + fk.ToColumn,
			Attrs: map[string]any{
				"from_table":  fk.FromTable,
				"from_column": fk.FromColumn,
				"to_table":    fk.ToTable,
				"to_column":   fk.ToColumn,
			},
			Evidence: ev,
		})
	}

	return facts
}
