// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/occurrence"
)

// Persistent occurrence-pattern editing (#1976). assembly.patternCreate now records the pattern
// (assembly_replication.go); these methods re-read it, suppress/unsuppress it as a whole or per
// element, reposition an element off the grid, and delete the whole array. Each edit drives the
// real occurrences the pattern generated, so suppressing an element hides its instance and
// deleting the pattern removes the copies (the seed stays).

// registerAssemblyPatternHandlers wires the pattern list + edit methods.
func (r *Router) registerAssemblyPatternHandlers() {
	r.readOnly(wire.MethodAssemblyPatternList, assemblyQuery(assemblyPatternList))
	r.mutating(wire.MethodAssemblyPatternSetSuppressed, "Suppress Pattern", typedAssembly(assemblyPatternSetSuppressed))
	r.mutating(wire.MethodAssemblyPatternElementSuppress, "Suppress Pattern Element", typedAssembly(assemblyPatternElementSetSuppressed))
	r.mutating(wire.MethodAssemblyPatternElementReposition, "Reposition Pattern Element", typedAssembly(assemblyPatternElementReposition))
	r.mutating(wire.MethodAssemblyPatternDelete, "Delete Pattern", typedAssembly(assemblyPatternDelete))
}

// assemblyPatternList re-reads every persistent pattern in the active assembly.
func assemblyPatternList(_ *app.Session, asm *compdef.AssemblyComponentDefinition) (wire.PatternListResult, error) {
	set := asm.Patterns()
	out := make([]wire.PatternInfo, set.Count())
	for i := 0; i < set.Count(); i++ {
		out[i] = patternInfo(set.Item(i))
	}
	return wire.PatternListResult{Patterns: out}, nil
}

// assemblyPatternSetSuppressed suppresses or unsuppresses a whole pattern.
func assemblyPatternSetSuppressed(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.SetPatternSuppressedArgs) (wire.PatternInfo, error) {
	pat, err := patternByID(asm, in.Pattern, wire.MethodAssemblyPatternSetSuppressed)
	if err != nil {
		return wire.PatternInfo{}, err
	}
	pat.SetSuppressed(in.Suppressed)
	return patternInfo(pat), nil
}

// assemblyPatternElementSetSuppressed suppresses or unsuppresses one element of a pattern.
func assemblyPatternElementSetSuppressed(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.SetPatternElementSuppressedArgs) (wire.PatternInfo, error) {
	pat, err := patternByID(asm, in.Pattern, wire.MethodAssemblyPatternElementSuppress)
	if err != nil {
		return wire.PatternInfo{}, err
	}
	if err := pat.SetElementSuppressed(in.Element, in.Suppressed); err != nil {
		return wire.PatternInfo{}, fmt.Errorf("%s: %w", wire.MethodAssemblyPatternElementSuppress, err)
	}
	return patternInfo(pat), nil
}

// assemblyPatternElementReposition moves one element of a pattern to an explicit placement.
func assemblyPatternElementReposition(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.RepositionPatternElementArgs) (wire.PatternInfo, error) {
	pat, err := patternByID(asm, in.Pattern, wire.MethodAssemblyPatternElementReposition)
	if err != nil {
		return wire.PatternInfo{}, err
	}
	if err := pat.RepositionElement(in.Element, matrixFromWire(in.Transform)); err != nil {
		return wire.PatternInfo{}, fmt.Errorf("%s: %w", wire.MethodAssemblyPatternElementReposition, err)
	}
	return patternInfo(pat), nil
}

// assemblyPatternDelete deletes a whole pattern and the occurrences it generated.
func assemblyPatternDelete(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.DeletePatternArgs) (wire.DeletePatternResult, error) {
	if _, err := patternByID(asm, in.Pattern, wire.MethodAssemblyPatternDelete); err != nil {
		return wire.DeletePatternResult{}, err
	}
	asm.Patterns().Delete(in.Pattern)
	return wire.DeletePatternResult{Deleted: in.Pattern}, nil
}

// patternByID resolves a pattern by session id, erroring with the method name when it is unknown.
func patternByID(asm *compdef.AssemblyComponentDefinition, id uint64, method string) (*occurrence.OccurrencePattern, error) {
	pat, ok := asm.Patterns().ByID(id)
	if !ok {
		return nil, fmt.Errorf("%s: no pattern with id %d", method, id)
	}
	return pat, nil
}
