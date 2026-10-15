// Package authorization proves whether an authorization check actually gates a
// handler's response, as opposed to merely being called.
//
// This is the distinction the whole project turns on. Finding a call to
// IsAdmin or CanAccess in a handler is a grep; it says nothing about safety. A
// check protects the response only when its failing result forces the request
// to abort before the response body is written — when the check's failure
// branch dominates the response write. A check whose result is thrown away, or
// evaluated on only some paths to the response, protects nothing while looking
// like protection, which is worse than no check at all.
//
// The analysis is SSA dominance over one handler. It is framework-agnostic: the
// caller supplies a predicate that recognizes its framework's response-writing
// calls, so this package never imports an adapter.
package authorization

import (
	"go/token"

	"golang.org/x/tools/go/ssa"
)

// Enforcement classifies how an authorization check relates to the response.
type Enforcement string

const (
	// Enforced: the check is branched on and its failure branch aborts before
	// every response write — the response is unreachable unless the check
	// passed.
	Enforced Enforcement = "enforced"

	// Discarded: the check's result is never used. The call runs; nothing
	// depends on it.
	Discarded Enforcement = "discarded"

	// ResultNotGating: the result is used but never reaches a branch, so it
	// cannot stop the request.
	ResultNotGating Enforcement = "result_not_gating"

	// NonDominating: the result is branched on, but the response write is
	// reachable without the check having been evaluated — the guard sits on
	// only some paths to the response.
	NonDominating Enforcement = "non_dominating"
)

// Enforced reports whether this enforcement value means the check protects the
// response.
func (e Enforcement) OK() bool { return e == Enforced }

// Check is one authorization-check call site and the verdict on whether it
// gates the handler's response.
type Check struct {
	Func        string // qualified name of the check function, e.g. "main.IsAdmin"
	Pos         token.Position
	Enforcement Enforcement
}

// CheckEnforcement finds every call in fn to a function named in checkFuncs and
// decides, for each, whether it gates the response. isResponseWrite recognizes
// the framework's response-body calls (for Gin, c.JSON and friends).
//
// Only fn's own body is examined. A check delegated to a helper the handler
// calls is not followed yet; that is a deliberate v1 narrowing, not an oversight.
func CheckEnforcement(fn *ssa.Function, checkFuncs map[string]bool, isResponseWrite func(*ssa.Call) bool) []Check {
	if fn == nil {
		return nil
	}

	// The blocks that write a response body. If a handler writes none, there is
	// nothing for a check to guard, and enforcement questions do not arise.
	writeBlocks := map[*ssa.BasicBlock]bool{}
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			if call, ok := instr.(*ssa.Call); ok && isResponseWrite(call) {
				writeBlocks[b] = true
			}
		}
	}

	var checks []Check
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			call, ok := instr.(*ssa.Call)
			if !ok {
				continue
			}
			callee := call.Call.StaticCallee()
			if callee == nil {
				continue
			}
			name := qualifiedName(callee)
			if !checkFuncs[name] {
				continue
			}
			checks = append(checks, Check{
				Func:        name,
				Pos:         fn.Prog.Fset.Position(call.Pos()),
				Enforcement: classify(call, writeBlocks),
			})
		}
	}
	return checks
}

// HasOwnershipGuard reports whether fn aborts the response on a branch that
// tests an identity-derived value, on every path to the response write. This is
// the general form of an ownership check: `if !CanAccessStudent(user, s)` and
// the inline `if user.OrgID != row.OrgID` both qualify, because both branch on a
// value derived from the authenticated principal and dominate the write. It is
// what distinguishes an IDOR from a correctly-guarded handler.
//
// isIdentity reports whether a value is derived from the principal (provenance).
func HasOwnershipGuard(fn *ssa.Function, isResponseWrite func(*ssa.Call) bool, isIdentity func(ssa.Value) bool) bool {
	if fn == nil {
		return false
	}
	writeBlocks := map[*ssa.BasicBlock]bool{}
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			if call, ok := instr.(*ssa.Call); ok && isResponseWrite(call) {
				writeBlocks[b] = true
			}
		}
	}
	if len(writeBlocks) == 0 {
		return false
	}
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			iff, ok := instr.(*ssa.If)
			if !ok {
				continue
			}
			if condUsesIdentity(iff.Cond, isIdentity) &&
				dominatesAllWrites(iff.Block(), writeBlocks) &&
				hasAbortingSuccessor(iff.Block(), writeBlocks) {
				return true
			}
		}
	}
	return false
}

// condUsesIdentity reports whether an identity-derived value feeds a branch
// condition, walking back through the expression that produced it.
func condUsesIdentity(cond ssa.Value, isIdentity func(ssa.Value) bool) bool {
	seen := map[ssa.Value]bool{}
	var walk func(v ssa.Value, depth int) bool
	walk = func(v ssa.Value, depth int) bool {
		if v == nil || depth > 12 || seen[v] {
			return false
		}
		seen[v] = true
		if isIdentity(v) {
			return true
		}
		if instr, ok := v.(ssa.Instruction); ok {
			for _, op := range instr.Operands(nil) {
				if op != nil && *op != nil && walk(*op, depth+1) {
					return true
				}
			}
		}
		return false
	}
	return walk(cond, 0)
}

// classify decides how one check call relates to the response writes.
func classify(call *ssa.Call, writeBlocks map[*ssa.BasicBlock]bool) Enforcement {
	refs := call.Referrers()
	if refs == nil || len(*refs) == 0 {
		return Discarded
	}

	ifBlock := controllingIf(call)
	if ifBlock == nil {
		return ResultNotGating
	}

	// The response is protected only if every write is dominated by the branch
	// on the check AND one side of that branch aborts without reaching a write.
	if dominatesAllWrites(ifBlock, writeBlocks) && hasAbortingSuccessor(ifBlock, writeBlocks) {
		return Enforced
	}
	return NonDominating
}

// controllingIf returns the block whose terminating If branches on the check's
// result, following the value through the boolean operations it may pass
// through first (a negation, a comparison, a logical combination). It returns
// nil when the result never reaches a branch.
func controllingIf(result ssa.Value) *ssa.BasicBlock {
	seen := map[ssa.Value]bool{result: true}
	queue := []ssa.Value{result}
	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		refs := v.Referrers()
		if refs == nil {
			continue
		}
		for _, instr := range *refs {
			if iff, ok := instr.(*ssa.If); ok {
				return iff.Block()
			}
			// Any value derived from the result (negation, comparison, phi)
			// might be the one the If tests; follow it.
			if derived, ok := instr.(ssa.Value); ok && !seen[derived] {
				seen[derived] = true
				queue = append(queue, derived)
			}
		}
	}
	return nil
}

// dominatesAllWrites reports whether every response-write block is dominated by
// ifBlock — i.e. no response write is reachable without first passing through
// the branch on the check. With no writes the condition is vacuously true.
func dominatesAllWrites(ifBlock *ssa.BasicBlock, writeBlocks map[*ssa.BasicBlock]bool) bool {
	for wb := range writeBlocks {
		if !ifBlock.Dominates(wb) {
			return false
		}
	}
	return true
}

// hasAbortingSuccessor reports whether at least one successor of ifBlock cannot
// reach any response write — the branch where a failed check aborts the request.
// Without it, both branches fall through to the response and the check gates
// nothing even while dominating the write.
func hasAbortingSuccessor(ifBlock *ssa.BasicBlock, writeBlocks map[*ssa.BasicBlock]bool) bool {
	for _, succ := range ifBlock.Succs {
		if !reachesWrite(succ, writeBlocks) {
			return true
		}
	}
	return false
}

// reachesWrite reports whether a response write is reachable from start over
// successor edges.
func reachesWrite(start *ssa.BasicBlock, writeBlocks map[*ssa.BasicBlock]bool) bool {
	seen := map[*ssa.BasicBlock]bool{}
	stack := []*ssa.BasicBlock{start}
	for len(stack) > 0 {
		b := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[b] {
			continue
		}
		seen[b] = true
		if writeBlocks[b] {
			return true
		}
		stack = append(stack, b.Succs...)
	}
	return false
}

// qualifiedName renders a function as package.Function, matching the names the
// config and the Gin adapter use (e.g. "main.IsAdmin").
func qualifiedName(fn *ssa.Function) string {
	if fn.Pkg != nil && fn.Pkg.Pkg != nil {
		return fn.Pkg.Pkg.Name() + "." + fn.Name()
	}
	return fn.Name()
}
