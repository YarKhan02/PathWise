package analyzer

import (
	"strings"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// FunctionsByName indexes the program's functions by the "package.Function"
// name the route facts use (e.g. "main.GetStudent", "course.GetCourse"), so a
// handler or middleware recovered by name in Phase 1 can be resolved to the SSA
// function Phase 2 and 3 walk. The naming matches the Gin adapter's
// qualifiedName, keeping the phases joined on the same identity.
//
// Methods are included, not just package-level functions: real handlers are
// methods on a Handler struct (course.GetCourse), and indexing only package
// members would leave every such handler unresolvable, so reachability would
// start from nothing. Synthetic wrappers (bound-method thunks, named with "$")
// are skipped; they are never a route handler.
//
// shortcut: the key drops the receiver type, so two methods of the same name in
// one package collide (last wins). Domain handlers have distinct names per
// package, so this is not hit in practice; revisit if a collision ever matters.
func FunctionsByName(prog *ssa.Program) map[string]*ssa.Function {
	out := map[string]*ssa.Function{}
	for fn := range ssautil.AllFunctions(prog) {
		if fn.Pkg == nil || fn.Pkg.Pkg == nil {
			continue
		}
		if strings.Contains(fn.Name(), "$") {
			continue
		}
		out[fn.Pkg.Pkg.Name()+"."+fn.Name()] = fn
	}
	return out
}
