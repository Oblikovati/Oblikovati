// SPDX-License-Identifier: GPL-2.0-only

package feature

import "oblikovati.org/model/param"

// Edit Feature scalar coverage for the assembly-context machining features (M11-F08,
// Oblikovati/Oblikovati#725): the assemblyFeatures.edit wire surface renders whatever
// EditableParams exposes, so a kind without an implementation cannot be edited after
// placement. The sketch-profiled kinds drive their scalar through a closure
// ([scalarParam]) so an edit reflows the next recompute; the box cut and the drilled
// hole bake a fixed tool at construction and so expose nothing editable here.

var (
	_ Editable = (*AssemblyExtrudeFeature)(nil)
	_ Editable = (*AssemblyRevolveFeature)(nil)
	_ Editable = (*AssemblySweepFeature)(nil)
)

// EditableParams exposes the assembly extrude's depth.
func (f *AssemblyExtrudeFeature) EditableParams() []EditableParam {
	return []EditableParam{scalarParam("Distance", param.Length, &f.distance)}
}

// EditableParams exposes the assembly revolve's sweep angle.
func (f *AssemblyRevolveFeature) EditableParams() []EditableParam {
	return []EditableParam{scalarParam("Angle", param.Angle, &f.angle)}
}

// EditableParams exposes the assembly sweep's total twist angle — the sibling of the extrude's
// Distance and the revolve's Angle, closing the #1648 gap so a placed assembly sweep edits in
// place (double-click / assemblyFeatures.edit) instead of forcing delete-and-recreate. The profile
// and explicit path are geometric inputs, not scalars, so they are not edited through this surface.
func (f *AssemblySweepFeature) EditableParams() []EditableParam {
	return []EditableParam{scalarParam("Twist", param.Angle, &f.twist)}
}
