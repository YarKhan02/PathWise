// Package analyzer builds the call graph and answers reachability questions
// over it: from a route's handler and middleware, what database access does the
// request actually reach, and which call sites could not be resolved.
//
// The call graph is CHA refined by VTA, the approach Go's own vulncheck uses.
// CHA (class hierarchy analysis) builds a sound but coarse graph from the type
// hierarchy alone — every call through an interface points at every type that
// implements it. VTA (variable type analysis) then prunes that graph using how
// values actually flow, collapsing most interface calls to the one concrete
// target they can reach. Pointer analysis would be the alternative, but
// go/pointer was removed from the toolchain (golang/go#59676), so this is the
// supported path.
package analyzer

import (
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/callgraph/vta"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// BuildCallGraph returns a call graph for prog, CHA refined once by VTA.
//
// Only the functions whose bodies the loader built (the application's own code)
// carry call edges; calls into dependencies land on target nodes without
// out-edges, which is exactly what reachability wants — it stops at the library
// boundary rather than walking into the standard library.
func BuildCallGraph(prog *ssa.Program) *callgraph.Graph {
	initial := cha.CallGraph(prog)
	return vta.CallGraph(ssautil.AllFunctions(prog), initial)
}
