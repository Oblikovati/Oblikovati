// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"slices"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/transform"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/assembly"
	"oblikovati.org/model/occurrence"
)

// Interference analysis (M12-F05, Oblikovati/Oblikovati#362/#368): where two placed components
// occupy the same space. The host owns the body geometry, so the computation lives here — each
// occurrence's bodies are transformed to world, broad-phased by bounding box, and (for
// overlapping pairs) boolean-intersected to measure the real overlap volume. The measurement
// integrates the intersection's ANALYTIC B-rep (ops.BodyGeometryProperties, M48/C3 #3451): a
// tessellated integral under-reports every curved overlap by ~π²/(3N²), and a bore-in-a-boss
// interference is exactly the curved case an assembly check exists to catch.

// interferenceVolumeEps ignores negligible overlaps (touching faces, numerical noise). It is a
// VOLUME gate on an already-measured lump, not a geometric distance, so it does not derive from
// geom.Resolution. // tol:volume
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
	for i := range occs {
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
		if _, _, found := bodiesOverlapVolume(moved, occurrenceWorldBodies(p)); found {
			return true // found already excludes a negligible (touching-face) overlap
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
		world, err := transform.TransformBody(b, o.Transform(), func(l topo.Lineage) topo.Lineage { return l })
		if err == nil && world != nil {
			out = append(out, world)
		}
	}
	return out
}

// bodiesOverlapVolume boolean-intersects every body pair and returns the total overlap volume
// (cm³) and a representative point, or ok=false when every overlap is negligible.
func bodiesOverlapVolume(as, bs []*topo.Body) (float64, math.Point3, bool) {
	var overlap overlapSum
	for _, ba := range as {
		for _, bb := range bs {
			inter, err := ops.Boolean(ops.Intersect, ba, bb)
			if err != nil || inter == nil {
				continue
			}
			overlap.fold(ops.BodyGeometryProperties(inter, ops.PropertyQuality()))
		}
	}
	return overlap.volume, overlap.centroid(), overlap.volume > 0
}

// overlapSum accumulates the analytic mass properties of every intersecting body-pair lump, so a
// multi-body occurrence pair reports ONE volume and ONE representative point. The point is the
// volume-weighted centroid of the overlap region — a quantity of the same analytic integral that
// yields the volume — replacing the first lump's bounding-box centre, which measured no lump but
// the first and, for an L-shaped or annular overlap, named a point outside the overlap entirely
// (M48/C3, Oblikovati/Oblikovati#3451).
type overlapSum struct {
	volume float64
	moment math.Vector3 // ∑ volume · centroid, about the origin
}

// fold adds one intersection body's properties, dropping a negligible lump (a touching-face
// contact) so numerical noise never creates an interference or drags the centroid.
func (s *overlapSum) fold(p ops.GeometryProperties) {
	if p.Volume <= interferenceVolumeEps {
		return
	}
	s.volume += p.Volume
	s.moment = s.moment.Add(p.Centroid.AsVector().Scale(math.Scalar(p.Volume)))
}

// centroid is the volume-weighted centre of the folded lumps, or the origin when none was folded
// (the caller reports no interference in that case, so the point is never read).
func (s *overlapSum) centroid() math.Point3 {
	if s.volume == 0 {
		return math.P3(0, 0, 0)
	}
	return s.moment.Scale(math.Scalar(1 / s.volume)).AsPoint()
}

// pairInSubset reports whether the pair is in scope: both ids in subset, or subset empty.
func pairInSubset(subset []uint64, a, b uint64) bool {
	if len(subset) == 0 {
		return true
	}
	return idInSet(subset, a) && idInSet(subset, b)
}

func idInSet(ids []uint64, v uint64) bool {
	return slices.Contains(ids, v)
}
