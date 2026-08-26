// SPDX-License-Identifier: GPL-2.0-only

package gopherlua

import (
	"context"

	lua "github.com/yuin/gopher-lua"

	"oblikovati.org/script"
)

// runDeadline derives the effective wall-clock deadline for one run from the caller's
// context and the Wall limit. gopher-lua's context-aware main loop checks ctx.Done()
// every opcode (vm.go mainLoopWithContext), so a cancellable context is what
// deterministically unwinds a runaway like `while true do end` — it is the primary
// runaway guard this VM can enforce per-opcode. The returned cancel MUST be called.
func runDeadline(parent context.Context, lim script.Limits) (context.Context, context.CancelFunc) {
	if lim.Wall <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, lim.Wall)
}

// applyMemoryLimits bounds the VM's data and call stacks at state creation, the only
// hard memory ceiling gopher-lua offers without its os.Exit-on-breach SetMx watchdog
// (which we deliberately do NOT use — it would crash the host, violating ADR-0028 §1).
// A non-positive MemBytes leaves the library defaults in place.
//
// MemBytes is translated to a registry size (each registry slot is one LValue); the
// call stack is kept proportionally bounded so deep recursion errors rather than grows
// unbounded. This is best-effort: the wall-clock guard backstops a true allocation bomb.
func memoryOptions(lim script.Limits) lua.Options {
	opts := lua.Options{SkipOpenLibs: true}
	if lim.MemBytes <= 0 {
		return opts
	}
	const bytesPerSlot = 32 // conservative LValue footprint estimate
	slots := max(int(lim.MemBytes/bytesPerSlot), minRegistrySize)
	opts.RegistrySize = slots
	opts.RegistryMaxSize = slots // hard cap: registry cannot grow past the budget
	opts.CallStackSize = callStackFor(slots)
	return opts
}

const (
	minRegistrySize  = 1024
	maxCallStackSize = 4096
)

// callStackFor sizes the call stack proportionally to the registry budget, clamped so
// a small memory cap still allows useful recursion and a large one stays bounded.
func callStackFor(slots int) int {
	cs := max(slots/16, 256)
	if cs > maxCallStackSize {
		cs = maxCallStackSize
	}
	return cs
}

// instructionBudget is the seam for an opcode-budget guard. gopher-lua exposes no
// public opcode hook, so the instruction count cannot be enforced directly on this VM;
// the wall-clock context is the enforced runaway guard (runDeadline). This function
// records the budget for metrics (Result.Op) and is the single place a future hooked VM
// (or a swapped Engine) would wire a real opcode counter — keeping Limits.Instructions
// a live part of the contract per the plan (ADR-0028).
func instructionBudget(lim script.Limits) uint64 { return lim.Instructions }
