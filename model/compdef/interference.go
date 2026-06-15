// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/assembly"
	"oblikovati.org/model/occurrence"
)

// Interference analysis (M12-F05, Oblikovati/Oblikovati#362/#368): where two placed components
// occupy the same space. The host owns the body geometry, so the computation lives here — each
// occurrence's bodies are transformed to world, broad-phased by bounding box, and (for
// overlapping pairs) boolean-intersected to measure the real overlap volume.

// interferenceVolumeEps ignores negligible overlaps (touching faces, numerical noise).
const interferenceVolumeEps = 1e-6

// AnalyzeInterference reports the overlapping volumes between the assembly's occurrences.
// subset (occurrence ids) restricts the analysis to pairs both within it; empty analyzes every
// pair.
func (a *AssemblyComponentDefinition) AnalyzeInterference(subset []uint64) assembly.InterferenceResults {
	occs := placedLeafOccurrences(a.occurrences)
	world := make(map[uint64][]*topo.Body, len(occs))
	boxes := make(map[uint64]math.Box, len(occs))
	for _, o := range occs {
		world[o.ID()] = occurrenceWorldBodies(o)
		boxes[o.ID()] = o.RangeBox()
	}
	var out assembly.InterferenceResults
	for i := 0; i < len(occs); i++ {
		for j := i + 1; j < len(occs); j++ {
			oa, ob := occs[i], occs[j]
			if !pairInSubset(subset, oa.ID(), ob.ID()) || !boxes[oa.ID()].Intersects(boxes[ob.ID()]) {
				continue
			}
			if vol, center, ok := bodiesOverlapVolume(world[oa.ID()], world[ob.ID()]); ok {
				out.Results = append(out.Results, assembly.InterferenceResult{A: oa.ID(), B: ob.ID(), Vol: vol, Center: center})
				out.Total += vol
			}
		}
	}
	return out
}

// WouldContactBlock reports whether moving the occurrence with occID to newTransform would make
// it interpenetrate one of its contact-set partners (M12-F05) — the contact solver's
// stop-at-contact test for a candidate move. It is false (no block) when the solver is
// disabled or the occurrence shares no contact set. The probe is non-destructive: the
// occurrence's transform is restored before returning.
func (a *AssemblyComponentDefinition) WouldContactBlock(occID uint64, newTransform math.Matrix4) bool {
	if !a.contacts.Enabled() {
		return false
	}
	partners := a.contacts.PartnersOf(occID)
	o, ok := a.occurrences.ByID(occID)
	if len(partners) == 0 || !ok {
		return false
	}
	saved := o.Transform()
	o.SetTransform(newTransform)
	defer o.SetTransform(saved)
	moved := occurrenceWorldBodies(o)
	for _, pid := range partners {
		p, ok := a.occurrences.ByID(pid)
		if !ok {
			continue
		}
		if vol, _, found := bodiesOverlapVolume(moved, occurrenceWorldBodies(p)); found && vol > interferenceVolumeEps {
			return true
		}
	}
	return false
}

// placedLeafOccurrences returns the non-suppressed occurrences that instance a part with
// bodies (sub-assembly interference is a later refinement).
func placedLeafOccurrences(occs *occurrence.Occurrences) []*occurrence.Occurrence {
	var out []*occurrence.Occurrence
	for _, o := range occs.All() {
		if o.Suppressed() {
			continue
		}
		if _, ok := o.Definition().(interface {
			SurfaceBodies() *topo.SurfaceBodies
		}); ok {
			out = append(out, o)
		}
	}
	return out
}

// occurrenceWorldBodies transforms an occurrence's part bodies into assembly space.
func occurrenceWorldBodies(o *occurrence.Occurrence) []*topo.Body {
	part, ok := o.Definition().(interface {
		SurfaceBodies() *topo.SurfaceBodies
	})
	if !ok {
		return nil
	}
	var out []*topo.Body
	for _, b := range part.SurfaceBodies().All() {
		world, err := ops.TransformBody(b, o.Transform(), func(l topo.Lineage) topo.Lineage { return l })
		if err == nil && world != nil {
			out = append(out, world)
		}
	}
	return out
}

// bodiesOverlapVolume boolean-intersects every body pair and returns the total overlap volume
// (cm³) and a representative point, or ok=false when the overlap is negligible.
func bodiesOverlapVolume(as, bs []*topo.Body) (float64, math.Point3, bool) {
	total := 0.0
	var center math.Point3
	found := false
	for _, ba := range as {
		for _, bb := range bs {
			inter, err := ops.Boolean(ops.Intersect, ba, bb)
			if err != nil || inter == nil {
				continue
			}
			v := bodyVolume(inter)
			if v <= interferenceVolumeEps {
				continue
			}
			total += v
			if !found {
				center = inter.RangeBox().Center()
				found = true
			}
		}
	}
	return total, center, found
}

// bodyVolume is the absolute enclosed volume of a body (sum of its shells' signed volumes).
func bodyVolume(b *topo.Body) float64 {
	v := 0.0
	for _, sh := range b.Shells() {
		v += ops.ShellSignedVolume(sh, ops.DefaultQuality())
	}
	if v < 0 {
		v = -v
	}
	return v
}

// pairInSubset reports whether the pair is in scope: both ids in subset, or subset empty.
func pairInSubset(subset []uint64, a, b uint64) bool {
	if len(subset) == 0 {
		return true
	}
	return idInSet(subset, a) && idInSet(subset, b)
}

func idInSet(ids []uint64, v uint64) bool {
	for _, id := range ids {
		if id == v {
			return true
		}
	}
	return false
}
