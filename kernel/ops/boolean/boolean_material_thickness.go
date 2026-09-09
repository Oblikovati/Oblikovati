// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"cmp"
	stdmath "math"
	"slices"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A solid body's THINNEST material, measured from the body's own boundary faces.
//
// It used to be the smallest side of the axis-aligned bounding box, which measures the coordinate
// frame as much as the body. Measured on the RING corpus pair (boolean_material_thickness_test.go):
// the same 1e-10 drill read 2e-10 thick along Z, 3.4043798713069418 thick after a 37 degree turn
// about (1,1,0) and 2.0000046063728405e-10 after 90 degrees — so the size classification REFUSED it
// at 0 and 90 degrees and BUILT it at 37 (#3524). A user who rotates a part and gets a different
// answer has no reason to trust any answer.
//
// The measure here is a minimum over the body's own analytic face pairs on one surface class, which
// turns with the body: planar faces bound slabs, a closed curved face wraps a rod, a ball or a tube,
// and nested cylinders or spheres hold a wall between them. The pair vocabulary and every geometry
// switch live in kernel/geom (geom.AsMaterialSlab, geom.EnclosedSpan, geom.OpposedSpan).

// solidThickness is a solid body's smallest material thickness — how thin its material gets in the
// direction it is thinnest, from its own faces, so the answer is the same for the body in any
// orientation. ok is false for a nil, non-solid or empty body, and for a body whose faces hold no
// pair with a closed-form thickness: a sheet has no material to measure, and an unproven thickness
// must not be reported as a thin one.
//
// ceiling is the width at which the measurement stops caring: the caller (a size classification)
// only asks whether the material is BELOW the model's weld, so a direction that already reaches the
// weld need not be measured to the end. A truncated direction always reports at least ceiling, so it
// can never become the minimum while a genuinely thinner one exists, and the value a refusal names is
// therefore always one that was measured in full. Pass 0 for the exact minimum: the width scan is
// then O(directions x vertices) with nothing cut short, and the difference grows with the body —
// 14.8 vs 8.65 microseconds on a 96-gon prism (98 faces, 192 vertices), 1.63 ms vs 0.23 ms on a
// 2000-gon one (2002 faces, 4000 vertices). Reading the bounding box used to cost 3.8 nanoseconds;
// that is the price of the answer being about the body rather than about the frame.
//
// Example:
//
//	if t, ok := solidThickness(tool, res.Weld()); ok && !res.Resolves(t) { /* refuse */ }
func solidThickness(b *topo.Body, ceiling float64) (float64, bool) {
	if b == nil || !b.IsSolid() {
		return 0, false
	}
	slabs, curved := boundarySurfacesOf(b.Faces())
	var thin spanFloor
	thin.offer(minCurvedSpan(curved))
	thin.offer(minSlabSpan(slabs, b.Vertices(), thin.ceiling(ceiling)))
	return thin.value, thin.found
}

// curvedBoundary is one non-planar face reduced to what a material measurement needs: its oriented
// surface, and whether its trim covers the whole surface (only then does what the surface encloses
// describe the body's material there).
type curvedBoundary struct {
	surface geom.BoundarySurface
	wrapped bool
}

// boundarySurfacesOf splits a body's faces into the planar slabs and the curved boundaries, each in
// the body's own face order so the measurement is byte-identical run to run.
func boundarySurfacesOf(faces []*topo.Face) ([]geom.MaterialSlab, []curvedBoundary) {
	slabs := make([]geom.MaterialSlab, 0, len(faces))
	var curved []curvedBoundary
	for _, f := range faces {
		bs := geom.BoundarySurface{Surface: f.Geometry(), Reversed: f.Reversed()}
		if slab, ok := geom.AsMaterialSlab(bs); ok {
			slabs = append(slabs, slab)
			continue
		}
		curved = append(curved, curvedBoundary{surface: bs, wrapped: faceWrapsItsSurface(f)})
	}
	return slabs, curved
}

// faceWrapsItsSurface reports whether a face's trim covers its surface completely, so the material
// the surface encloses is really the material the body has there. Two shapes count: a face with no
// boundary at all (a whole torus), and one that uses an edge TWICE — the seam that makes a periodic
// face simply connected, whose loop runs up the seam, around, and back down it (brep.SolidCylinder).
//
// Without this test a fillet's narrow cylindrical strip would report its own small radius as the
// body's thickness, and a 1e-9 fillet on a 10-thick plate would be refused as sub-resolution.
func faceWrapsItsSurface(f *topo.Face) bool {
	loops := f.Loops()
	if len(loops) == 0 {
		return true
	}
	return slices.ContainsFunc(loops, loopCrossesASeam)
}

// loopCrossesASeam reports whether one loop uses any edge more than once.
func loopCrossesASeam(l *topo.Loop) bool {
	seen := make(map[*topo.Edge]bool)
	for _, u := range l.EdgeUses() {
		if seen[u.Edge()] {
			return true
		}
		seen[u.Edge()] = true
	}
	return false
}

// minSlabSpan is the body's smallest width across a pair of opposed planar boundaries. Sorting by
// direction turns what would be a test of every planar face against every other — a
// hundred-thousand-facet import makes that quadratic — into one linear pass per direction group.
func minSlabSpan(slabs []geom.MaterialSlab, verts []*topo.Vertex, ceiling float64) (float64, bool) {
	slices.SortFunc(slabs, compareSlabs)
	var thin spanFloor
	for lo := 0; lo < len(slabs); {
		hi := slabGroupEnd(slabs, lo)
		thin.offer(slabGroupWidth(slabs[lo:hi], verts, thin.ceiling(ceiling)))
		lo = hi
	}
	return thin.value, thin.found
}

// compareSlabs is the total order the grouping needs: direction first (so parallel faces are
// adjacent), then offset along it, then the material side. Every key is compared exactly, and the
// order is strict, so the arrangement — and the answer — is the same on every platform.
func compareSlabs(a, b geom.MaterialSlab) int {
	// Written as early returns rather than cmp.Or, which evaluates every one of its arguments before
	// choosing between them — five float compares and their NaN tests per comparison. The sort is the
	// measurement's largest single cost on a many-faced body, so the difference is worth the five ifs.
	if c := cmp.Compare(a.Dir.X(), b.Dir.X()); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Dir.Y(), b.Dir.Y()); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Dir.Z(), b.Dir.Z()); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Offset, b.Offset); c != 0 {
		return c
	}
	return cmp.Compare(boolOrder(a.MaterialAbove), boolOrder(b.MaterialAbove))
}

// boolOrder ranks the two material sides so compareSlabs is a STRICT total order: no two distinct
// slabs compare equal, so an unstable sort still produces one arrangement, on every platform.
func boolOrder(b bool) int {
	if b {
		return 1
	}
	return 0
}

// slabGroupEnd returns the end of the run of slabs parallel to the one at lo. Each candidate is
// compared with the run's FIRST member rather than its predecessor, so the grouping cannot drift.
func slabGroupEnd(slabs []geom.MaterialSlab, lo int) int {
	hi := lo + 1
	for hi < len(slabs) && slabs[lo].SameDirection(slabs[hi]) {
		hi++
	}
	return hi
}

// slabGroupWidth is the body's own WIDTH across one pair of opposed planar boundaries: the group has
// to hold a face with material above it AND one with material below, and the width is then the body's
// whole extent along that direction.
//
// It is the body's extent and not the two faces' own separation on purpose. The faces' separation is
// a LOCAL gap, and on a non-convex or multi-lump body it reads the space between two unrelated
// regions as a thickness: measured against the NopSCADlib corpus, the nearest-pair form reported a
// star washer as −0.5988740122992224 thick and an IDC transition as 6.938893903907228e-18, neither of
// which is a thing about the body. The extent turns with the body just as well, because the DIRECTION
// comes from the body's own faces.
//
// The extent is taken over the body's vertices, which is exact for a planar-bounded body and can only
// UNDER-report where a curved face bulges past every vertex — a body's curved thinness is measured by
// EnclosedSpan and OpposedSpan instead.
func slabGroupWidth(group []geom.MaterialSlab, verts []*topo.Vertex, ceiling float64) (float64, bool) {
	if !groupBracketsMaterial(group) {
		return 0, false
	}
	return vertexExtentAlong(group[0].Dir, verts, ceiling)
}

// groupBracketsMaterial reports whether a direction group holds faces on BOTH sides of the material —
// one whose material lies above it and one whose material lies below. A group with only one side is a
// set of steps, not a thickness.
func groupBracketsMaterial(group []geom.MaterialSlab) bool {
	above, below := false, false
	for _, s := range group {
		above = above || s.MaterialAbove
		below = below || !s.MaterialAbove
	}
	return above && below
}

// vertexExtentAlong is how far the body reaches along dir. ok is false for a body with no vertices
// (a whole torus or sphere, whose thickness EnclosedSpan answers) and for a degenerate zero extent.
func vertexExtentAlong(dir math.UnitVector3, verts []*topo.Vertex, ceiling float64) (float64, bool) {
	if len(verts) == 0 {
		return 0, false
	}
	lo, hi := stdmath.Inf(1), stdmath.Inf(-1)
	for _, v := range verts {
		d := float64(v.Point().AsVector().Dot(dir.AsVector()))
		lo, hi = stdmath.Min(lo, d), stdmath.Max(hi, d)
		if hi-lo >= ceiling {
			break // this direction cannot hold the minimum the caller is looking for
		}
	}
	return hi - lo, hi > lo
}

// minCurvedSpan is the thinnest material the body's curved faces bound: what a closed face wraps, and
// what a nested same-class pair holds between it.
func minCurvedSpan(curved []curvedBoundary) (float64, bool) {
	var thin spanFloor
	for i, c := range curved {
		if c.wrapped {
			thin.offer(geom.EnclosedSpan(c.surface))
		}
		thin.offer(minOpposedSpan(c.surface, curved[i+1:]))
	}
	return thin.value, thin.found
}

// minOpposedSpan is the thinnest wall between one curved boundary and any of the others.
func minOpposedSpan(s geom.BoundarySurface, rest []curvedBoundary) (float64, bool) {
	var thin spanFloor
	for _, o := range rest {
		thin.offer(geom.OpposedSpan(s, o.surface))
	}
	return thin.value, thin.found
}

// spanFloor accumulates the smallest span offered to it, and remembers whether anything was: an
// unmeasured body must be told apart from a zero-thickness one.
type spanFloor struct {
	value float64
	found bool
}

// offer keeps v when it is proven and thinner than everything offered so far.
func (f *spanFloor) offer(v float64, ok bool) {
	if ok && (!f.found || v < f.value) {
		f.value, f.found = v, true
	}
}

// ceiling is the width beyond which a further measurement cannot change the answer: the caller's own
// ceiling, or the thinnest span found so far when that is smaller. A ceiling of 0 or less means the
// caller wants the exact minimum and nothing is truncated.
func (f *spanFloor) ceiling(callers float64) float64 {
	if callers <= 0 {
		callers = stdmath.Inf(1)
	}
	if f.found && f.value < callers {
		return f.value
	}
	return callers
}
