// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/assembly"
)

// The assembly contact & interference surface (M12-F05, #362/#368): manage contact sets
// (occurrences that resist interpenetration), toggle the contact solver, and run a static
// interference analysis reporting the overlapping volumes between occurrences.

// registerContactHandlers wires the contactSets.*/contactSolver.*/interference.* methods.
func (r *Router) registerContactHandlers() {
	r.readOnly(wire.MethodContactSetsCreate, contactSetsCreate)
	r.readOnly(wire.MethodContactSetsList, contactSetsList)
	r.readOnly(wire.MethodContactSetsDelete, contactSetsDelete)
	r.readOnly(wire.MethodContactSetsAddMember, contactSetsAddMember)
	r.readOnly(wire.MethodContactSetsRemoveMember, contactSetsRemoveMember)
	r.readOnly(wire.MethodContactSolverSetEnabled, contactSolverSetEnabled)
	r.readOnly(wire.MethodContactSolverStatus, contactSolverStatus)
	r.readOnly(wire.MethodInterferenceAnalyze, interferenceAnalyze)
}

func contactSetsCreate(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.CreateContactSetArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	cs := asm.ContactSolver().Create(in.Name)
	return json.Marshal(wire.ContactSetResult{ContactSet: contactSetInfo(cs)})
}

func contactSetsList(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	out := make([]wire.ContactSetInfo, 0)
	for _, cs := range asm.ContactSolver().All() {
		out = append(out, contactSetInfo(cs))
	}
	return json.Marshal(wire.ContactSetsResult{ContactSets: out})
}

func contactSetsDelete(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.ContactSetRef
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	asm.ContactSolver().Delete(in.ID)
	return contactSetsList(s, nil)
}

func contactSetsAddMember(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	return contactMember(s, raw, (*assembly.ContactSolver).AddMember)
}

func contactSetsRemoveMember(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	return contactMember(s, raw, (*assembly.ContactSolver).RemoveMember)
}

// contactMember adds or removes a member via mutate and returns the updated set.
func contactMember(s *app.Session, raw json.RawMessage, mutate func(*assembly.ContactSolver, uint64, uint64) error) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.ContactMemberArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if err := mutate(asm.ContactSolver(), in.Set, in.Occurrence); err != nil {
		return nil, err
	}
	return json.Marshal(wire.ContactSetResult{ContactSet: contactSetInfo(asm.ContactSolver().ByID(in.Set))})
}

func contactSolverSetEnabled(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.ContactSolverEnableArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	asm.ContactSolver().SetEnabled(in.Enabled)
	return contactSolverStatus(s, nil)
}

func contactSolverStatus(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	sv := asm.ContactSolver()
	return json.Marshal(wire.ContactSolverResult{Solver: wire.ContactSolverInfo{Enabled: sv.Enabled(), SetCount: sv.SetCount()}})
}

func interferenceAnalyze(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.AnalyzeInterferenceArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	res := asm.AnalyzeInterference(in.Occurrences)
	out := make([]wire.InterferenceResultInfo, 0, len(res.Results))
	for _, r := range res.Results {
		out = append(out, interferenceResultInfo(r))
	}
	return json.Marshal(wire.InterferenceResultsResult{Results: out, TotalVolume: res.Total})
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
