// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/assembly"
	"oblikovati.org/model/compdef"
)

// The assembly contact & interference surface (M12-F05, #362/#368): manage contact sets
// (occurrences that resist interpenetration), toggle the contact solver, and run a static
// interference analysis reporting the overlapping volumes between occurrences.

// registerContactHandlers wires the contactSets.*/contactSolver.*/interference.* methods.
func (r *Router) registerContactHandlers() {
	r.mutating(wire.MethodContactSetsCreate, "Create Contact Set", typedAssembly(contactSetsCreate))
	r.readOnly(wire.MethodContactSetsList, assemblyQuery(contactSetsList))
	r.mutating(wire.MethodContactSetsDelete, "Delete Contact Set", typedAssembly(contactSetsDelete))
	r.mutating(wire.MethodContactSetsAddMember, "Edit Contact Set", typedAssembly(contactSetsAddMember))
	r.mutating(wire.MethodContactSetsRemoveMember, "Edit Contact Set", typedAssembly(contactSetsRemoveMember))
	r.readOnly(wire.MethodContactSolverSetEnabled, typedAssembly(contactSolverSetEnabled))
	r.readOnly(wire.MethodContactSolverStatus, assemblyQuery(contactSolverStatus))
	r.readOnly(wire.MethodInterferenceAnalyze, typedAssembly(interferenceAnalyze))
}

func contactSetsCreate(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.CreateContactSetArgs) (wire.ContactSetResult, error) {
	cs := asm.ContactSolver().Create(in.Name)
	return wire.ContactSetResult{ContactSet: contactSetInfo(cs)}, nil
}

func contactSetsList(_ *app.Session, asm *compdef.AssemblyComponentDefinition) (wire.ContactSetsResult, error) {
	return contactSetsListResult(asm), nil
}

// contactSetsListResult renders the active assembly's contact sets.
func contactSetsListResult(asm *compdef.AssemblyComponentDefinition) wire.ContactSetsResult {
	out := make([]wire.ContactSetInfo, 0)
	for _, cs := range asm.ContactSolver().All() {
		out = append(out, contactSetInfo(cs))
	}
	return wire.ContactSetsResult{ContactSets: out}
}

func contactSetsDelete(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.ContactSetRef) (wire.ContactSetsResult, error) {
	asm.ContactSolver().Delete(in.ID)
	return contactSetsListResult(asm), nil
}

func contactSetsAddMember(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.ContactMemberArgs) (wire.ContactSetResult, error) {
	return contactMember(asm, in, (*assembly.ContactSolver).AddMember)
}

func contactSetsRemoveMember(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.ContactMemberArgs) (wire.ContactSetResult, error) {
	return contactMember(asm, in, (*assembly.ContactSolver).RemoveMember)
}

// contactMember adds or removes a member via mutate and returns the updated set.
func contactMember(asm *compdef.AssemblyComponentDefinition, in wire.ContactMemberArgs, mutate func(*assembly.ContactSolver, uint64, uint64) error) (wire.ContactSetResult, error) {
	if err := mutate(asm.ContactSolver(), in.Set, in.Occurrence); err != nil {
		return wire.ContactSetResult{}, err
	}
	return wire.ContactSetResult{ContactSet: contactSetInfo(asm.ContactSolver().ByID(in.Set))}, nil
}

func contactSolverSetEnabled(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.ContactSolverEnableArgs) (wire.ContactSolverResult, error) {
	asm.ContactSolver().SetEnabled(in.Enabled)
	return contactSolverStatusResult(asm), nil
}

func contactSolverStatus(_ *app.Session, asm *compdef.AssemblyComponentDefinition) (wire.ContactSolverResult, error) {
	return contactSolverStatusResult(asm), nil
}

// contactSolverStatusResult renders the contact solver's enabled/set-count status.
func contactSolverStatusResult(asm *compdef.AssemblyComponentDefinition) wire.ContactSolverResult {
	sv := asm.ContactSolver()
	return wire.ContactSolverResult{Solver: wire.ContactSolverInfo{Enabled: sv.Enabled(), SetCount: sv.SetCount()}}
}

func interferenceAnalyze(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.AnalyzeInterferenceArgs) (wire.InterferenceResultsResult, error) {
	res := asm.AnalyzeInterference(in.Occurrences)
	out := make([]wire.InterferenceResultInfo, 0, len(res.Results))
	for _, r := range res.Results {
		out = append(out, interferenceResultInfo(r))
	}
	return wire.InterferenceResultsResult{Results: out, TotalVolume: res.Total}, nil
}

// contactSetInfo encodes a contact set.
func contactSetInfo(cs assembly.ContactSetView) wire.ContactSetInfo {
	return wire.ContactSetInfo{ID: cs.ID(), Name: cs.Name(), Members: cs.Members()}
}

// interferenceResultInfo encodes one interference result.
func interferenceResultInfo(r assembly.InterferenceResult) wire.InterferenceResultInfo {
	return wire.InterferenceResultInfo{
		OccurrenceA: r.OccurrenceA(),
		OccurrenceB: r.OccurrenceB(),
		Volume:      r.Volume(),
		Center:      types.Point{X: r.Center.X, Y: r.Center.Y, Z: r.Center.Z},
	}
}
