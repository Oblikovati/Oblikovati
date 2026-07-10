// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/occurrence"
	"oblikovati.org/model/sketch"
)

// Occurrence-qualified work-feature references (#1857): an assembly names a datum inside a
// sub-component occurrence with "occ/<path>/plane|axis|point/N". This file resolves such a
// reference to its geometry in the assembly's space — used both to accept the ref as a datum input
// (the ExternalDatumResolver the assembly installs on its WorkGeometry) and to list a component's
// datums as occurrence-qualified refs (ResolveOccurrenceContext, for the wire list).

// ResolveOccurrenceContext resolves an occurrence path (instance names, top-down) to the composed
// occurrence-to-assembly transform and the target component's work geometry. ok is false when the
// path names no occurrence, or the leaf occurrence is not a part (a sub-assembly has no datums to
// surface here).
func (a *AssemblyComponentDefinition) ResolveOccurrenceContext(pathNames []string) (math.Matrix4, *feature.WorkGeometry, bool) {
	chain, ok := a.Occurrences().ResolveChain(occurrence.OccurrencePath(pathNames))
	if !ok {
		return math.Identity4(), nil, false
	}
	transform := math.Identity4()
	for _, o := range chain {
		transform = transform.Mul(o.Transform()) // world = parent · child (assembly_derive.go)
	}
	part, ok := chain[len(chain)-1].Definition().(*PartComponentDefinition)
	if !ok {
		return math.Identity4(), nil, false
	}
	return transform, part.WorkGeometry(), true
}

// occurrenceDatumResolver adapts an assembly to feature.ExternalDatumResolver: it resolves an
// occurrence-qualified ref to a sub-component datum, transformed into the assembly's space (#1857).
type occurrenceDatumResolver struct{ asm *AssemblyComponentDefinition }

// context decodes ref and resolves its occurrence path to (transform, component work geometry,
// component-local native ref).
func (r occurrenceDatumResolver) context(ref feature.WorkRef) (math.Matrix4, *feature.WorkGeometry, feature.WorkRef, bool) {
	path, native, ok := feature.ParseOccurrenceRef(ref)
	if !ok {
		return math.Identity4(), nil, "", false
	}
	transform, work, ok := r.asm.ResolveOccurrenceContext(path)
	if !ok {
		return math.Identity4(), nil, "", false
	}
	return transform, work, native, true
}

// OccurrencePlane resolves an occurrence-qualified plane ref, transformed into assembly space.
func (r occurrenceDatumResolver) OccurrencePlane(ref feature.WorkRef) (sketch.Plane, bool) {
	transform, work, native, ok := r.context(ref)
	if !ok {
		return sketch.Plane{}, false
	}
	pl, err := work.ResolvePlaneRef(native)
	if err != nil {
		return sketch.Plane{}, false
	}
	return transformPlane(transform, pl)
}

// OccurrenceAxisLine resolves an occurrence-qualified axis ref to its origin + unit direction in
// assembly space.
func (r occurrenceDatumResolver) OccurrenceAxisLine(ref feature.WorkRef) (math.Point3, math.UnitVector3, bool) {
	transform, work, native, ok := r.context(ref)
	if !ok {
		return math.Point3{}, math.UnitVector3{}, false
	}
	wa, ok := work.AxisByRef(native)
	if !ok {
		return math.Point3{}, math.UnitVector3{}, false
	}
	dir, err := transform.TransformUnitVector(wa.Direction())
	if err != nil {
		return math.Point3{}, math.UnitVector3{}, false
	}
	return transform.TransformPoint(wa.Origin()), dir, true
}

// OccurrencePoint resolves an occurrence-qualified point ref, transformed into assembly space.
func (r occurrenceDatumResolver) OccurrencePoint(ref feature.WorkRef) (math.Point3, bool) {
	transform, work, native, ok := r.context(ref)
	if !ok {
		return math.Point3{}, false
	}
	p, err := work.ResolvePointRef(native)
	if err != nil {
		return math.Point3{}, false
	}
	return transform.TransformPoint(p), true
}

// transformPlane maps a sketch plane through a rigid occurrence transform: its origin and in-plane
// axes are transformed, preserving normal = XAxis × YAxis. ok is false if the transform degenerates
// an axis (a zero-scale placement).
func transformPlane(m math.Matrix4, pl sketch.Plane) (sketch.Plane, bool) {
	x, err := m.TransformUnitVector(pl.XAxis())
	if err != nil {
		return sketch.Plane{}, false
	}
	y, err := m.TransformUnitVector(pl.YAxis())
	if err != nil {
		return sketch.Plane{}, false
	}
	out, err := sketch.NewPlane(m.TransformPoint(pl.Origin()), x, y)
	if err != nil {
		return sketch.Plane{}, false
	}
	return out, true
}
