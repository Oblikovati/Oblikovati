// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/occurrence"
)

// An assembly definition is a body source a part can derive from: it flattens its
// occurrence tree into placed bodies (M11-F06). It already reports its geometry
// version, so a derived part re-derives when the assembly changes.
var _ feature.AssemblyBodySource = (*AssemblyComponentDefinition)(nil)

// PlacedBodies flattens the assembly's occurrence tree into the bodies of its leaf
// parts, each paired with the world transform that places it (composed down the
// occurrence path) and the occurrence it belongs to. Suppressed occurrences are
// skipped. It is what a derived-assembly component pulls to build a base body — and
// what shrinkwrap will simplify (M11-F06).
func (a *AssemblyComponentDefinition) PlacedBodies() []feature.PlacedBody {
	var out []feature.PlacedBody
	collectPlacedBodies(a.occurrences, math.Identity4(), nil, &out, nil)
	return out
}

// collectPlacedBodies walks one occurrence level, composing each occurrence's transform
// with its parent's and extending the instance-name path, emitting a leaf part's bodies
// and recursing into sub-assemblies. The path disambiguates a shared flyweight reached
// through several placements (M11-F08 nested participation). flexParent, when non-nil, is the
// flexible sub-assembly occurrence whose children we are walking: each child uses that
// placement's independent override transform when one is set, so two placements of one flexible
// sub-assembly show their components in different positions (M12-F06).
func collectPlacedBodies(occs *occurrence.Occurrences, parent math.Matrix4, path occurrence.OccurrencePath, out *[]feature.PlacedBody, flexParent *occurrence.Occurrence) {
	for _, o := range occs.All() {
		if o.Suppressed() {
			continue
		}
		local := o.Transform()
		if flexParent != nil {
			if override, ok := flexParent.ChildTransform(o.Name()); ok {
				local = override
			}
		}
		world := parent.Mul(local)
		here := append(append(occurrence.OccurrencePath(nil), path...), o.Name())
		switch def := o.Definition().(type) {
		case bodyDefinition: // a leaf part: emit its bodies placed in the assembly
			for _, b := range def.SurfaceBodies().All() {
				*out = append(*out, feature.PlacedBody{Body: b, Transform: world, Source: o, Path: here})
			}
		case occurrence.Composite: // a sub-assembly: recurse, carrying flexible-child overrides
			var nextFlex *occurrence.Occurrence
			if o.Flexible() {
				nextFlex = o
			}
			collectPlacedBodies(def.Occurrences(), world, here, out, nextFlex)
		}
	}
}

// bodyDefinition is a component definition that owns evaluated bodies — a part
// definition. Matched structurally so the flatten works for any body-bearing leaf.
type bodyDefinition interface {
	SurfaceBodies() *topo.SurfaceBodies
}
