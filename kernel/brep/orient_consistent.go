// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// orientFaceSigns derives a CONSISTENT outward orientation sign (±1) for each prepared face, INDEPENDENT
// of the stored Face.Reversed flag. The winding number needs every face signed so its normal points out
// of the body; an imported B-rep's Reversed flags can be inconsistent (a STEP extrusion→quadric normal-
// side defect), which makes the signed solid angles cancel and a solid interior read as outside.
//
// The reliable orientation carrier on a valid B-rep is the LOOP traversal (STEP ORIENTED_EDGE), not the
// face normal-side flag: on a consistently-oriented shell every shared edge is walked in opposite
// directions by its two faces, so each face's outer-loop handedness about its geometric normal (S_u×S_v)
// already defines one global orientation. The sign of the outer loop's signed area in (u, v) is that
// handedness (+1 CCW about S_u×S_v → normal is outward, −1 → inward).
//
// The one bit a handedness cannot see — which of the two consistent colourings is outward — is decided
// PER CONNECTED SHELL, not once for the face set: a body can carry several shells, and each seeds its
// own colouring. A shell no other shell encloses is turned to integrate a positive volume; a shell
// enclosed by one already oriented is a VOID, whose material lies outside it, and integrates negative.
// The mixed boolean's two-lump result (a pin cut clean through) made the single global flip visible:
// the sum of a positive lump and an inverted one stayed positive, so the inverted lump was never turned
// (ADR-0060). A one-shell body takes exactly the path it always did.
func orientFaceSigns(faces []fluxFace) []float64 {
	signs := make([]float64, len(faces))
	for i := range faces {
		signs[i] = loopHandedness(faces[i].region)
	}
	var oriented []fluxFace
	for _, shell := range fluxShellsLargestFirst(faces, signs) {
		enclosed := shellEnclosedBy(oriented, faces, shell)
		probed := probeShellSigns(faces, shell, enclosed)
		bit := shellBit(faces, signs, shell, probed)
		for _, i := range shell {
			if probed[i] != 0 {
				signs[i] = probed[i]
			} else {
				signs[i] *= bit
			}
			f := faces[i]
			f.sign = signs[i]
			oriented = append(oriented, f)
		}
	}
	return signs
}

// probeShellSigns reads each face's outward sign from the geometry, ORIENTATION-FREE: a point a
// stand-off along the face's geometric normal S_u×S_v, and one against it, are classified by ray
// parity against the shell (an odd crossing count is inside — no sign is read). The material of a
// lump is inside its shell, that of a void outside, so the sign is +1 when the +normal side is the
// non-material side. A face whose two probes agree (a wall thinner than the stand-off) or whose rays
// all graze reads 0, and falls back to its loop handedness under the shell's bit.
//
// This replaced reading the sign from the loop handedness alone: on the stubs a near-pinch cut leaves,
// the handedness read two of a stub's three faces wrong, and the whole-body volume sign that used to
// settle the last bit hid it because the errors balanced (ADR-0060).
func probeShellSigns(faces []fluxFace, shell []int, enclosed bool) map[int]float64 {
	own := newShellProbe(fluxFacesAt(faces, shell))
	step := float64(own.box.Diagonal().Length()) * probeOffsetRel
	out := map[int]float64{}
	for _, i := range shell {
		q, n, ok := faceProbePoint(&faces[i])
		if !ok {
			continue
		}
		plus, okP := own.parityInside(q.TranslateBy(n.Scale(math.Scalar(step))))
		minus, okM := own.parityInside(q.TranslateBy(n.Scale(math.Scalar(-step))))
		if !okP || !okM || plus == minus {
			continue
		}
		if plus == enclosed {
			out[i] = 1
		} else {
			out[i] = -1
		}
	}
	return out
}

// shellBit is the global sign the handedness readings of a shell need: the majority verdict of the
// faces the probe could read, else the sign that makes the shell's own volume positive (a lump) or
// negative (a void) — the one-shell rule the whole body always used.
func shellBit(faces []fluxFace, signs []float64, shell []int, probed map[int]float64) float64 {
	agree := 0.0
	for _, i := range shell {
		agree += probed[i] * signs[i]
	}
	if agree != 0 {
		return stdmath.Copysign(1, agree)
	}
	if shellSignedVolume(faces, signs, shell) < 0 {
		return -1
	}
	return 1
}

// probeOffsetRel is how far off a face the traversal probe sits, as a fraction of the shell's diagonal:
// well clear of the weld tolerance, well inside any feature a valid solid has at that scale.
const probeOffsetRel = 1e-3 // tol:numeric — probe stand-off as a fraction of the shell diagonal (dimensionless)

// faceProbePoint returns an in-trim point of the face, the one farthest from its trim boundary on a
// coarse (u,v) grid, and the unit geometric normal S_u×S_v there.
func faceProbePoint(f *fluxFace) (math.Point3, math.Vector3, bool) {
	best, bestDist, found := math.Point2{}, -1.0, false
	for i := range volumeGridSteps {
		for j := range volumeGridSteps {
			uv := math.P2(f.u0+(f.u1-f.u0)*(float64(i)+0.5)/volumeGridSteps, f.v0+(f.v1-f.v0)*(float64(j)+0.5)/volumeGridSteps)
			if !f.region.contains(uv) {
				continue
			}
			if d := f.region.boundaryDistance(uv); d > bestDist {
				best, bestDist, found = uv, d, true
			}
		}
	}
	if !found {
		return math.Point3{}, math.Vector3{}, false
	}
	du, dv := f.cf.surface.DerivativesAt(float64(best.X), float64(best.Y))
	n, err := math.UnitVector3FromVector(du.Cross(dv))
	if err != nil {
		return math.Point3{}, math.Vector3{}, false
	}
	return f.cf.surface.PointAt(float64(best.X), float64(best.Y)), n.AsVector(), true
}

// shellEnclosedBy reports whether a shell is a VOID of the shells already oriented outward: its box
// lies within theirs and one of its points classifies inside them. The box gate is what keeps two
// lumps that TOUCH — a rod's stubs either side of a near-pinch cut, whose boundaries pass within the
// tolerance of each other — from being classified at a point the flux cannot read.
func shellEnclosedBy(oriented, faces []fluxFace, shell []int) bool {
	if len(oriented) == 0 {
		return false
	}
	box := paddedBox(fluxFacesBox(oriented), probeOffsetRel)
	if !box.ContainsBox(fluxFacesBox(fluxFacesAt(faces, shell))) {
		return false
	}
	q := &fluxQuery{faces: oriented}
	return q.inside(shellSamplePoint(faces, shell), box)
}

// fluxFacesAt selects the prepared faces of one shell.
func fluxFacesAt(faces []fluxFace, shell []int) []fluxFace {
	out := make([]fluxFace, 0, len(shell))
	for _, i := range shell {
		out = append(out, faces[i])
	}
	return out
}

// fluxFacesBox bounds the prepared faces: their loop vertices and their in-trim surface samples on
// the coarse (u,v) grid — a vertex box alone misses a rim circle's sweep, which has one vertex.
func fluxFacesBox(faces []fluxFace) math.Box {
	box := curvedFaceBox(curvedFacesOf(faces))
	for i := range faces {
		f := &faces[i]
		for a := range volumeGridSteps + 1 {
			for b := range volumeGridSteps + 1 {
				uv := math.P2(f.u0+(f.u1-f.u0)*float64(a)/volumeGridSteps, f.v0+(f.v1-f.v0)*float64(b)/volumeGridSteps)
				if f.region.contains(uv) {
					box = box.ExtendPoint(f.cf.surface.PointAt(float64(uv.X), float64(uv.Y)))
				}
			}
		}
	}
	return box
}

// paddedBox grows a box by a fraction of its diagonal on every side.
func paddedBox(box math.Box, rel float64) math.Box {
	pad := float64(box.Diagonal().Length()) * rel
	d := math.V3(pad, pad, pad)
	return math.Box{Min: box.Min.TranslateBy(d.Scale(-1)), Max: box.Max.TranslateBy(d)}
}

// fluxShellsLargestFirst groups the faces into connected shells over their shared edges, ordered by
// decreasing enclosed volume so a container is oriented before anything it may enclose.
func fluxShellsLargestFirst(faces []fluxFace, signs []float64) [][]int {
	cfs := curvedFacesOf(faces)
	pw := newWelder3(geom.ResolutionForBox(curvedFaceBox(cfs)).Stitch())
	shells := connectedFaceComponents(curvedFaceAdjacency(cfs, pw))
	sort.SliceStable(shells, func(a, b int) bool {
		return stdmath.Abs(shellSignedVolume(faces, signs, shells[a])) > stdmath.Abs(shellSignedVolume(faces, signs, shells[b]))
	})
	return shells
}

// shellSignedVolume is the shell's enclosed volume under the given per-face signs.
func shellSignedVolume(faces []fluxFace, signs []float64, shell []int) float64 {
	volume := 0.0
	for _, i := range shell {
		volume += signs[i] * faceVolumeTerm(&faces[i])
	}
	return volume
}

// shellSamplePoint is a point ON the shell (a loop vertex of its first face), for the enclosure test
// against the shells oriented before it.
func shellSamplePoint(faces []fluxFace, shell []int) math.Point3 {
	f := faces[shell[0]].cf
	if len(f.loops) == 0 || len(f.loops[0].edges) == 0 {
		return f.surface.PointAt(faces[shell[0]].u0, faces[shell[0]].v0)
	}
	return f.loops[0].edges[0].start()
}

// fluxWindingAt sums each prepared face's signed solid angle at p: a closed outward shell gives ≈4π
// inside and ≈0 outside.
func fluxWindingAt(faces []fluxFace, p math.Point3) float64 {
	total := 0.0
	for i := range faces {
		f := &faces[i]
		total += f.sign * integrateFluxCell(f.cf.surface, p, f.region, f.u0, f.u1, f.v0, f.v1, 0)
	}
	return total
}

// loopHandedness is the sign of the outer ring's signed area in (u, v): +1 when the ring runs CCW about
// the surface normal S_u×S_v (so that normal is the outward one), −1 when CW — NEGATED when the face is
// the ring's COMPLEMENT, because a ring that runs CCW around the region it encloses runs CW around
// everything else on the domain. Without that negation the big spherical cap of a ball joined with a rod
// took the inward normal as outward, and the ball's own centre read outside its solid
// (Oblikovati/Oblikovati#3453, #3429). A boundaryless face (a whole sphere/torus, its own closed body)
// has no ring to read; it returns +1 and leans on the volume sign.
func loopHandedness(r trimRegion) float64 {
	if len(r.rings) == 0 {
		return 1
	}
	sign := 1.0
	if regionSignedArea(r) < 0 {
		sign = -1
	}
	if r.complement {
		return -sign
	}
	return sign
}

// regionSignedArea is the signed area the face's rings enclose in (u, v).
//
// A ring that closes in the parameter plane contributes its own shoelace, and they are SUMMED: a hole
// is wound against its enclosing ring, so it subtracts, and the total keeps the outer ring's sign
// without having to decide which ring is outer.
//
// A ring that does NOT close there is a band rim — a closed circuit in 3-D, but an open polyline in the
// covering space. Its shoelace says nothing: a rim at constant v shoelaces to ZERO whichever way the
// band is oriented, which is what made a plate's bore wall report the handedness of its own opposite
// and integrate the bore as added rather than subtracted, 564.36 against a true 354.92. Those rims are
// joined into ONE circuit before the area is taken. A band is bounded by its two rims and the two seam
// segments, and concatenating the rims lets the shoelace supply both seams itself: the junction between
// the rims is one, the closing chord of the concatenation is the other.
//
// This is what OCCT gets for free. It stores a seam edge EXPLICITLY — twice in the wire, with two
// pcurves — so a band's wire is already a closed contour and ShapeAnalysis::TotCross2D applies to it
// unchanged. Our bands carry no seam edge, so the circuit is reassembled here
// (Oblikovati/Oblikovati#3506).
func regionSignedArea(r trimRegion) float64 {
	area := 0.0
	var wrapping []math.Point2
	for _, ring := range r.rings {
		if !ringSpansAPeriod(ring, r) {
			area += signedArea2D(ring)
			continue
		}
		wrapping = append(wrapping, ring...)
	}
	if len(wrapping) < 3 {
		return area
	}
	return area + signedArea2D(wrapping)
}

// ringSpansAPeriod reports whether a ring travels a WHOLE TURN in a periodic parameter instead of
// returning to where it started — what makes it an open polyline in the covering space. A face on a
// non-periodic surface can never do it, and neither can a ring that walks out along a seam and back
// (a full cone's side loop), whose net travel is zero.
func ringSpansAPeriod(ring []math.Point2, r trimRegion) bool {
	if len(ring) < 3 {
		return false
	}
	if r.uPeriodic && ringClosesByAWholeTurn(ring, func(p math.Point2) float64 { return float64(p.X) }) {
		return true
	}
	return r.vPeriodic && ringClosesByAWholeTurn(ring, func(p math.Point2) float64 { return float64(p.Y) })
}

// ringClosesByAWholeTurn reports whether the ring's CLOSING chord — from its last sample back to its
// first, the one edge the sample list leaves implicit — jumps a whole turn rather than one sampling
// step.
//
// It is judged against the ring's OWN widest interior step, so it assumes nothing about how densely
// the ring was sampled. A rim's closing chord is the whole turn less one step, which is many times the
// widest step; a ring that genuinely closes has a closing chord the size of its neighbours. Comparing
// against a fixed fraction of the period instead would misread a sparsely sampled patch that happens
// to reach far in u.
func ringClosesByAWholeTurn(ring []math.Point2, at func(math.Point2) float64) bool {
	closing, widest := stdmath.Abs(at(ring[len(ring)-1])-at(ring[0])), 0.0
	for i := 1; i < len(ring); i++ {
		widest = stdmath.Max(widest, stdmath.Abs(at(ring[i])-at(ring[i-1])))
	}
	return closing > twoPi/2 && closing > 2*widest
}

// twoPi is one full turn, the period of every angular surface parameter in the kernel.
const twoPi = 2 * stdmath.Pi

// volumeGridSteps is the fixed (u, v) sampling per face for the signed-volume sign probe. The probe needs
// only the SIGN of the body volume to fix the global orientation, so a coarse uniform grid — no adaptive
// refinement, no singularity to resolve — is enough.
const volumeGridSteps = 8

// faceVolumeTerm is the face's contribution to 3·V, the divergence-theorem volume ∫∫ P·(S_u×S_v) du dv
// over its trimmed domain (with the loop-handedness sign applied by the caller). A coarse uniform grid
// weighted by the in-trim fraction; only its sign, summed over the body, is used.
func faceVolumeTerm(f *fluxFace) float64 {
	du := (f.u1 - f.u0) / volumeGridSteps
	dv := (f.v1 - f.v0) / volumeGridSteps
	sum := 0.0
	for i := 0; i < volumeGridSteps; i++ {
		for j := 0; j < volumeGridSteps; j++ {
			u0, v0 := f.u0+float64(i)*du, f.v0+float64(j)*dv
			frac := cellTrimFraction(f.region, u0, u0+du, v0, v0+dv)
			if frac == 0 {
				continue
			}
			sum += volumeIntegrand(f.cf.surface, u0+0.5*du, v0+0.5*dv) * du * dv * frac
		}
	}
	return sum
}

// volumeIntegrand evaluates P·(S_u×S_v) at (u, v) — the outward flux of the position field, whose surface
// integral is three times the enclosed volume (Gauss divergence of F = P).
func volumeIntegrand(s geom.Surface, u, v float64) float64 {
	du, dv := s.DerivativesAt(u, v)
	n := du.Cross(dv)
	p := s.PointAt(u, v)
	return float64(p.AsVector().Dot(n))
}

// curvedFacesOf strips the prepared faces back to their curvedFaces.
func curvedFacesOf(faces []fluxFace) []curvedFace {
	cfs := make([]curvedFace, len(faces))
	for i, f := range faces {
		cfs[i] = f.cf
	}
	return cfs
}

// shellProbe is one shell prepared for repeated parity casts: each face's box, so a ray tests only the
// faces it can reach, instead of developing every face's trim for every cast — which made the probe
// quadratic in the face count and stalled a fine-pitch coil join.
type shellProbe struct {
	faces []curvedFace
	boxes []math.Box
	// bands holds each face's polygon-vs-curve boundary error, measured ONCE. It is a pure function
	// of the face and the ray traversal needs it per (face, ray); recomputing it there — it walks
	// every trim edge measuring chord sagitta — made it a quarter of a shell orientation pass, which
	// casts thousands of rays at the same face set (#3459).
	bands []float64
	box   math.Box
}

// newShellProbe boxes every face of the shell, each padded by a share of its own diagonal so the
// sampled box can never miss a curved face's true extent between samples (a 45° sample step on a
// rim leaves a 7.6% sagitta; the pad is 10%).
func newShellProbe(faces []fluxFace) *shellProbe {
	p := &shellProbe{faces: curvedFacesOf(faces), boxes: make([]math.Box, len(faces)), bands: make([]float64, len(faces)), box: fluxFacesBox(faces)}
	for i := range faces {
		p.boxes[i] = paddedBox(fluxFacesBox(faces[i:i+1]), probeBoxPadRel)
		p.bands[i] = faceBoundaryBand(p.faces[i])
	}
	return p
}

// probeBoxPadRel pads a face's sampled box for the ray cull: above the sagitta a 9-sample grid leaves
// on a full rim (7.6% of the diagonal), so the cull is conservative.
const probeBoxPadRel = 0.1 // tol:numeric — box cull pad as a fraction of the face diagonal (dimensionless)

// parityInside classifies p by crossing parity along a clean ray, testing only the faces whose box
// the ray enters (rayParityInsideClean's verdict, culled).
func (s *shellProbe) parityInside(p math.Point3) (inside, ok bool) {
	return firstCleanDirection(s.box, func(dir [3]float64, tMax, tol float64) (bool, bool) {
		d := math.V3(dir[0], dir[1], dir[2])
		ray, err := geom.NewLine(p, d)
		if err != nil {
			return false, false
		}
		crossings := 0
		for i, f := range s.faces {
			if _, hits := s.boxes[i].IntersectsRay(p, d); !hits {
				continue
			}
			n, clean := faceRayCrossingsBand(f, ray, tMax, s.bands[i], tol)
			if !clean {
				return false, false
			}
			crossings += n
		}
		return crossings%2 == 1, true
	})
}
