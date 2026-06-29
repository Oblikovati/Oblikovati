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
// incremental parameter-edit fast path (#1414) while an assembly does a full re-solve — each
// holder picks its own strategy behind the same contract.
//
// It deliberately carries NO persistence (MarshalSnapshot/RecipeContent): undo recording is a
// separate concern, composed by the caller (see app's editable-holder seam), so this stays a
// minimal parameter abstraction.
//
//	h, ok := d.Content().(compdef.ParameterHolder) // true for a part OR an assembly
type ParameterHolder interface {
	Parameters() *param.Parameters
	Recompute()
	RecomputeAfterParameterEdit()
}

// Both component definitions are parameter holders. A plain interface assertion on a
// document's content then discriminates "has parameters" (part or assembly) from "does not"
// (e.g. a drawing) without a type switch.
var (
	_ ParameterHolder = (*PartComponentDefinition)(nil)
	_ ParameterHolder = (*AssemblyComponentDefinition)(nil)
)
