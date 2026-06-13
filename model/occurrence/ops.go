// SPDX-License-Identifier: GPL-2.0-only

package occurrence

import "oblikovati.org/math"

// Assembly-level replication ops (M11-F04, PBI-122): mirror, copy, and substitute
// operate on an [Occurrences] collection, sharing the placed definitions (the
// flyweight) and reusing the collection's id minting.

// MirrorComponents adds a mirror of each source occurrence to dst, reflected across
// the plane (origin, normal). Each mirror shares its source's definition but is placed
// by reflection·sourceTransform, so it is correctly handed: the reflection's
// determinant is -1, flipping the placement's orientation. Returns the new occurrences
// in source order.
//
// A chiral component is handed here purely by its placement transform; deriving an
// independent mirrored PART (a new, separately editable document) is M11-F06.
func MirrorComponents(dst *Occurrences, sources []*Occurrence, origin math.Point3, normal math.UnitVector3) []*Occurrence {
	reflect := math.Reflection4(origin, normal)
	out := make([]*Occurrence, 0, len(sources))
	for _, s := range sources {
		mirrored := reflect.Mul(s.Transform())
		out = append(out, dst.AddByComponentDefinition(s.name+"-mirror", s.definition, mirrored))
	}
	return out
}

// CopyComponents adds an independent copy of each source occurrence to dst — the same
// definition and placement, but a new, unlinked instance (unlike a pattern element,
// which tracks its seed). Returns the copies in source order.
func CopyComponents(dst *Occurrences, sources []*Occurrence) []*Occurrence {
	out := make([]*Occurrence, 0, len(sources))
	for _, s := range sources {
		out = append(out, dst.AddByComponentDefinition(s.name+"-copy", s.definition, s.transform))
	}
	return out
}

// Substitute swaps the source occurrences for a single substitute: it suppresses the
// sources (so they leave the model but can be restored) and adds one occurrence,
// flagged [Occurrence.IsSubstitute], that references simplified — a simplified
// representation of them placed at transform. Returns the substitute.
//
// The simplified definition is the caller's (a shrinkwrap/derived part generated in
// M11-F06); this is the substitution mechanism, not the geometry reduction.
func Substitute(dst *Occurrences, sources []*Occurrence, name string, simplified Definition, transform math.Matrix4) *Occurrence {
	for _, s := range sources {
		s.SetSuppressed(true)
	}
	sub := dst.AddByComponentDefinition(name, simplified, transform)
	sub.substitute = true
	return sub
}
