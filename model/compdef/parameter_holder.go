// SPDX-License-Identifier: GPL-2.0-only

package compdef

import "oblikovati.org/model/param"

// ParameterHolder is a document content that owns a parameter table and knows how to
// recompute itself after a parameter edit. Both a part and an assembly are holders, so a
// cross-document concern — Derived Parameter Tables (#605) — can resolve its target by this
// interface instead of casting to the concrete part type (M39-F02, #1558). It mirrors the
// reference API, where a DerivedParameterTable.Parent is a part OR an assembly component
// definition.
//
// The recompute methods are named by intent, not by mechanism: a part may take its
// incremental change fast path (#1414) while an assembly does a full re-solve — each holder
// picks its own strategy behind the same contract. RecomputeAfterChange is the single
// generalized invalidation seam (ADR-0044): it takes no change-set (it drains its own change
// sources) so the same entry serves a parameter edit today and a cross-part adaptive-reference
// change later, and a wholesale implementation (the assembly's) is a valid point on it.
//
// It deliberately carries NO persistence (MarshalSnapshot/RecipeContent): undo recording is a
// separate concern, composed by the caller (see app's editable-holder seam), so this stays a
// minimal parameter abstraction.
//
//	h, ok := d.Content().(compdef.ParameterHolder) // true for a part OR an assembly
type ParameterHolder interface {
	Parameters() *param.Parameters
	// Units are the holder's display units — a parameter value is formatted in them, so the
	// wire/UI parameter surface needs them alongside the table.
	Units() param.UnitsOfMeasure
	Recompute()
	RecomputeAfterChange()
}

// Both component definitions are parameter holders. A plain interface assertion on a
// document's content then discriminates "has parameters" (part or assembly) from "does not"
// (e.g. a drawing) without a type switch.
var (
	_ ParameterHolder = (*PartComponentDefinition)(nil)
	_ ParameterHolder = (*AssemblyComponentDefinition)(nil)
)
