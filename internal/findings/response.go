package findings

import (
	"fmt"
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/ssa"

	"github.com/YarKhan02/pathwise/internal/config"
	"github.com/YarKhan02/pathwise/internal/model"
)

// ResponseScope reports a handler that serializes a field backed by a column
// declared sensitive, on an endpoint whose declared scope does not include it —
// an over-broad response, even when the authorization to reach it is sound.
//
// It fires only when configured (which columns are sensitive, and what each
// endpoint may return, is policy). For each response body the handler writes, it
// reads the serialized struct's JSON field names and, for every sensitive
// `table.column` whose table this endpoint reads, flags a field named for that
// column that the endpoint's allow-list does not permit.
//
// bodyOf extracts the serialized value from a framework response call (for Gin,
// the object in c.JSON(status, obj)), keeping this rule framework-agnostic.
//
// shortcut: a field is matched to a column by name (its JSON tag), not by a
// resolved field→column mapping, so the confidence is "likely", not "proven".
func ResponseScope(scanID, endpoint string, handler *ssa.Function, tablesRead []string,
	cfg *config.Config, bodyOf func(*ssa.Call) (ssa.Value, bool)) []model.Finding {

	if handler == nil || !cfg.ScopeConfigured() || !cfg.ScopeDeclared(endpoint) {
		return nil // no sensitive columns configured, or no declared scope to exceed
	}
	reads := map[string]bool{}
	for _, t := range tablesRead {
		reads[strings.ToLower(t)] = true
	}
	allowed := map[string]bool{}
	for _, f := range cfg.AllowedFields(endpoint) {
		allowed[strings.ToLower(f)] = true
	}

	var out []model.Finding
	seen := map[string]bool{}
	for _, b := range handler.Blocks {
		for _, instr := range b.Instrs {
			call, ok := instr.(*ssa.Call)
			if !ok {
				continue
			}
			body, ok := bodyOf(call)
			if !ok {
				continue
			}
			fields := jsonFields(body)
			pos := handler.Prog.Fset.Position(call.Pos())
			for _, sc := range cfg.SensitiveColumns {
				table, col, ok := splitColumn(sc)
				if !ok || !reads[table] || !fields[col] {
					continue
				}
				qualified := table + "." + col
				if allowed[qualified] {
					continue // in this endpoint's declared scope
				}
				if seen[qualified] {
					continue
				}
				seen[qualified] = true
				out = append(out, model.Finding{
					ScanID:     scanID,
					Kind:       model.FindingResponseFieldExceedsScope,
					Severity:   model.SeverityMedium,
					Confidence: model.ConfidenceLikely,
					Endpoint:   endpoint,
					StableKey:  string(model.FindingResponseFieldExceedsScope) + "|" + endpoint + "|" + qualified,
					Title:      "Response exposes out-of-scope field " + qualified,
					Detail: fmt.Sprintf(
						"This endpoint serializes a %q field backed by %s, declared sensitive, which this endpoint's scope does not permit. The authorization may be sound; the response is over-broad.",
						col, qualified),
					Remedy:    fmt.Sprintf("Omit %s from the response, or add it to this endpoint's allow_fields if it is intended.", qualified),
					Locations: []model.SourceLocation{{File: pos.Filename, StartLine: pos.Line, EndLine: pos.Line}},
					Premises:  []string{"serialized field " + col + "; sensitive column " + qualified},
				})
			}
		}
	}
	return out
}

// jsonFields returns the set of JSON field names a serialized value exposes,
// descending through pointers and slices to the struct underneath. The value is
// the object passed to c.JSON, boxed into an interface, so it is unboxed first.
func jsonFields(v ssa.Value) map[string]bool {
	out := map[string]bool{}
	if mi, ok := v.(*ssa.MakeInterface); ok {
		v = mi.X
	}
	st := structOf(v.Type())
	if st == nil {
		return out
	}
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if !f.Exported() {
			continue
		}
		name := jsonName(reflect.StructTag(st.Tag(i)), f.Name())
		if name != "" && name != "-" {
			out[strings.ToLower(name)] = true
		}
	}
	return out
}

// structOf unwraps interface boxing, pointers and slices to the struct type a
// value ultimately carries, or nil if there is none.
func structOf(t types.Type) *types.Struct {
	for i := 0; i < 8 && t != nil; i++ { // bounded unwrap
		switch u := t.(type) {
		case *types.Pointer:
			t = u.Elem()
		case *types.Slice:
			t = u.Elem()
		case *types.Array:
			t = u.Elem()
		case *types.Named:
			if s, ok := u.Underlying().(*types.Struct); ok {
				return s
			}
			t = u.Underlying()
		case *types.Struct:
			return u
		default:
			return nil
		}
	}
	return nil
}

// jsonName returns the field's JSON name from its tag, or the Go field name
// when there is no json tag.
func jsonName(tag reflect.StructTag, fieldName string) string {
	j := tag.Get("json")
	if j == "" {
		return fieldName
	}
	if comma := strings.IndexByte(j, ','); comma >= 0 {
		j = j[:comma]
	}
	if j == "" {
		return fieldName
	}
	return j
}

func splitColumn(s string) (table, col string, ok bool) {
	i := strings.IndexByte(s, '.')
	if i < 0 {
		return "", "", false
	}
	return strings.ToLower(s[:i]), strings.ToLower(s[i+1:]), true
}
