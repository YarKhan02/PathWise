package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// EmbedIndex maps a package-level string variable to the SQL text embedded into
// it via a //go:embed directive. It exists because real projects keep each
// query in its own .sql file and pull it in with //go:embed, so the argument at
// a db.QueryContext call site is a string variable, not a literal. Without this,
// every such query reads as dynamic_sql and no table is recovered — a false
// "we cannot see this query" on code that is perfectly analyzable.
//
// Only string (and []byte) embeds are indexed, which Go requires to name a
// single file, so the mapping is one variable to one file's contents.
type EmbedIndex map[types.Object]string

// BuildEmbedIndex reads every //go:embed-backed string variable in the given
// packages and records the contents of the file each one embeds. A file that
// cannot be read is skipped, so its query falls back to dynamic_sql rather than
// to a wrong guess.
func BuildEmbedIndex(pkgs []*packages.Package) EmbedIndex {
	idx := EmbedIndex{}
	for _, pkg := range pkgs {
		if pkg.TypesInfo == nil || pkg.Fset == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			dir := filepath.Dir(pkg.Fset.Position(file.Pos()).Filename)
			for _, decl := range file.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.VAR {
					continue
				}
				for _, spec := range gd.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					// The //go:embed directive sits in the doc comment of the
					// spec, or of the decl when the var stands alone.
					pattern := embedPattern(vs.Doc, gd.Doc)
					if pattern == "" {
						continue
					}
					for _, name := range vs.Names {
						obj := pkg.TypesInfo.Defs[name]
						if obj == nil || !isStringType(obj.Type()) {
							continue
						}
						if text, ok := readEmbed(dir, pattern); ok {
							idx[obj] = text
						}
					}
				}
			}
		}
	}
	return idx
}

// embedPattern returns the single file pattern of the first //go:embed
// directive found in the given comment groups, or "" if none carry one.
func embedPattern(groups ...*ast.CommentGroup) string {
	for _, g := range groups {
		if g == nil {
			continue
		}
		for _, c := range g.List {
			rest, ok := strings.CutPrefix(c.Text, "//go:embed")
			if !ok {
				continue
			}
			fields := strings.Fields(rest)
			if len(fields) == 0 {
				continue
			}
			// A string embed names one file; take the first pattern and drop
			// any surrounding quotes.
			return strings.Trim(fields[0], "\"`")
		}
	}
	return ""
}

// readEmbed resolves pattern relative to dir and returns the file's contents.
// A glob is resolved to its first match, which is all a string embed can hold.
func readEmbed(dir, pattern string) (string, bool) {
	path := filepath.Join(dir, pattern)
	if strings.ContainsAny(pattern, "*?[") {
		matches, err := filepath.Glob(path)
		if err != nil || len(matches) == 0 {
			return "", false
		}
		path = matches[0]
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(data), true
}

// isStringType reports whether t is the string type.
func isStringType(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Kind() == types.String
}
