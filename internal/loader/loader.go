// Package loader turns a directory on disk into type-checked packages and an
// SSA program the analyzers consume. It is deliberately framework-agnostic: the
// Gin adapter, the call-graph builder and the dataflow analyzers all start from
// the same load, so loading happens once per scan rather than once per analyzer.
//
// The load is where the most common real-world failure surfaces. go/packages
// (and therefore SSA) requires the target to type-check; a missing dependency
// or an uncommitted generated file stops analysis before it begins. That is why
// Load returns the per-package errors instead of swallowing them: the caller
// turns them into build_error facts so a scan that analyzed nothing never reads
// as a scan that found nothing.
package loader

import (
	"fmt"
	"go/token"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// Loaded is the result of loading a target for analysis.
type Loaded struct {
	// Fset is shared by the type-checker and SSA, so a token.Pos from either
	// resolves against it.
	Fset *token.FileSet

	// Prog is the SSA program. Only the initial packages (Packages below) have
	// their function bodies built; dependencies are present as declarations so
	// that calls into them resolve to a target function, which is all route and
	// call-graph discovery needs. Building every dependency's bodies would pull
	// in the whole of a module like modernc.org/sqlite for no benefit.
	Prog *ssa.Program

	// Packages are the SSA packages for the target's own code, aligned with
	// Initial. An entry is nil when its go/packages counterpart failed to load.
	Packages []*ssa.Package

	// Initial are the type-checked packages matched by the load pattern.
	Initial []*packages.Package
}

// BuildErrors collects every error go/packages reported across the loaded
// packages. An empty slice means the target type-checked cleanly.
func (l *Loaded) BuildErrors() []packages.Error {
	var errs []packages.Error
	packages.Visit(l.Initial, nil, func(p *packages.Package) {
		errs = append(errs, p.Errors...)
	})
	return errs
}

// loadMode asks for exactly what SSA construction needs: names, files, the full
// import graph, and complete type information including syntax.
const loadMode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedCompiledGoFiles |
	packages.NeedImports |
	packages.NeedDeps |
	packages.NeedTypes |
	packages.NeedSyntax |
	packages.NeedTypesInfo |
	packages.NeedTypesSizes

// Load type-checks the packages matched by patterns, resolved relative to dir,
// and builds SSA for the matched packages. dir is the module to analyze; for
// the corpus apps this is their own module directory, not the PathWise root.
//
// patterns defaults to "./..." when none are given.
func Load(dir string, patterns ...string) (*Loaded, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	cfg := &packages.Config{
		Mode: loadMode,
		Dir:  dir,
		// Tests pull a package's _test.go into a second synthetic package,
		// which would double-count route registrations. Route discovery wants
		// the application as it is built, not its test variant.
		Tests: false,
	}

	initial, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", dir, err)
	}
	if len(initial) == 0 {
		return nil, fmt.Errorf("load %s: no packages matched %v", dir, patterns)
	}

	// Create SSA for the whole import closure so that a call into a dependency
	// resolves to a real *ssa.Function, then build bodies only for the target's
	// own packages. ssautil.AllPackages returns the SSA packages aligned with
	// initial; building just those is what keeps the load fast.
	prog, ssaPkgs := ssautil.AllPackages(initial, ssa.BuilderMode(0))
	for _, p := range ssaPkgs {
		if p != nil {
			p.Build()
		}
	}

	return &Loaded{
		Fset:     prog.Fset,
		Prog:     prog,
		Packages: ssaPkgs,
		Initial:  initial,
	}, nil
}
