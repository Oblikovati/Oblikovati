// SPDX-License-Identifier: GPL-2.0-only

// Package occurrence models component occurrences — the placements of a component (a
// part or a sub-assembly) inside an assembly. One referenced component can be placed
// many times; each placement is an occurrence carrying its own transform and
// suppression state. This is the assembly analogue of a body in a part: the unit the
// assembly's structure, range box, and (from M12) constraints operate on. It is the
// reference API's ComponentOccurrence / ComponentOccurrences (M11-F01, #345).
package occurrence

import "oblikovati.org/math"

// RangeBoxSource is what an occurrence instances: anything that reports its local
// (definition-space) range box. Both a part and an assembly component definition
// satisfy it (model/compdef), so an occurrence's contribution to its owner's box
// always reflects the current child geometry — and a sub-assembly can be placed in a
// parent assembly (nesting). Kept narrow here so model/occurrence does not import
// model/compdef, which imports this package.
type RangeBoxSource interface {
	// RangeBox returns the source's axis-aligned bounding box in its own space.
	RangeBox() math.Box
}

// Occurrence is one placement of a component inside an assembly: its source
// definition, a transform locating it in the assembly's space, and whether it is
// suppressed (excluded from the model). Created and owned by an [Occurrences]
// collection; mutating its transform or suppression bumps the owner's revision so the
// assembly's geometry version advances.
type Occurrence struct {
	id         uint64
	name       string
	transform  math.Matrix4
	suppressed bool
	source     RangeBoxSource
	owner      *Occurrences
}

// ID returns the occurrence's session id (unique within its owning collection).
func (o *Occurrence) ID() uint64 { return o.id }

// Name returns the occurrence's instance name (e.g. "pin:1").
func (o *Occurrence) Name() string { return o.name }

// Source returns the component definition this occurrence instances.
func (o *Occurrence) Source() RangeBoxSource { return o.source }

// Transform returns the occurrence's placement in its assembly's space.
func (o *Occurrence) Transform() math.Matrix4 { return o.transform }

// SetTransform repositions the occurrence and advances the owning assembly's version.
func (o *Occurrence) SetTransform(m math.Matrix4) {
	o.transform = m
	o.owner.bump()
}

// Suppressed reports whether the occurrence is excluded from the model — it
// contributes no geometry and is skipped by the range box (and, from M12, by
// constraint solving).
func (o *Occurrence) Suppressed() bool { return o.suppressed }

// SetSuppressed sets the occurrence's suppression state, advancing the version only
// when it actually changes.
func (o *Occurrence) SetSuppressed(suppressed bool) {
	if o.suppressed == suppressed {
		return
	}
	o.suppressed = suppressed
	o.owner.bump()
}

// RangeBox returns the occurrence's bounding box in its assembly's space: the
// source's local box placed by the transform. A suppressed occurrence reports the
// empty box, contributing nothing to the assembly.
func (o *Occurrence) RangeBox() math.Box {
	if o.suppressed {
		return math.EmptyBox()
	}
	return o.source.RangeBox().Transform(o.transform)
}
