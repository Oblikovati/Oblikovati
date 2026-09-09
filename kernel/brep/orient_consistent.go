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
//
// The geometric probe (probeShellSides) settles that ONE BIT and nothing else. It used to override a
// face's handedness outright wherever the two disagreed, and on a shell whose loops are one orientation
// that can only introduce error: the loops already agree with each other, so a single face reading
// differently is the PROBE misreading it — its stand-off is a fraction of the whole shell's diagonal,
// and the multipoint disk's Extrusion5 hands it a wall 0.6 µm across, where a 4 mm stand-off samples
// nothing near the face. The override then wrote a sense contradicting that face's own loops, and the
// boolean's emission gate (brep.FaceWindingConsistent) refused the body. A majority over the faces the
// probe COULD read is what a shell's free bit needs, and shellBit already takes it.
func orientFaceSigns(faces []fluxFace) []float64 {
	signs := make([]float64, len(faces))
	for i := range faces {
		signs[i] = loopHandedness(faces[i].cf, faces[i].region)
	}
	var oriented []fluxFace
	for _, shell := range fluxShellsLargestFirst(faces, signs) {
		sides := probeShellSides(faces, shell)
		probed := shellProbedSigns(sides, shellEnclosedBy(oriented, faces, shell, sides))
		bit := shellBit(faces, signs, shell, probed)
		for _, i := range shell {
			signs[i] *= bit
			f := faces[i]
			f.sign = signs[i]
			oriented = append(oriented, f)
		}
	}
	return signs
}

// shellSides is one shell read ORIENTATION-FREE, by its own ray parity: for each face the probe could
// decide, whether the +S_u×S_v side of it lies INSIDE the region the shell bounds, and one point that
// lies strictly inside that region. Both readings come from the same casts, so the enclosure decision
// and the per-face signs can never be taken from two different readings of the same shell.
type shellSides struct {
	plusInside map[int]bool
	interior   math.Point3
	found      bool // whether interior holds a point
}

// probeShellSides reads the shell's own geometry: a point a stand-off along each face's geometric
// normal S_u×S_v, and one against it, classified by ray parity against the shell (an odd crossing count
// is inside the region it bounds). A face whose two probes agree (a wall thinner than the stand-off) or
// whose rays all graze reads nothing and is passed over.
//
// This replaced reading a face's sign from its loop handedness alone: on the stubs a near-pinch cut
// leaves, the handedness read two of a stub's three faces wrong, and the whole-body volume sign that
// used to settle the last bit hid it because the errors balanced (ADR-0060). The INTERIOR point it also
// yields is what the enclosure test needs — see shellEnclosedBy.
func probeShellSides(faces []fluxFace, shell []int) shellSides {
	own := newShellProbe(fluxFacesAt(faces, shell))
	step := float64(own.box.Diagonal().Length()) * probeOffsetRel
	out := shellSides{plusInside: make(map[int]bool, len(shell))}
	for _, i := range shell {
		up, down, plus, ok := faceSideInShell(own, &faces[i], step)
		if !ok {
			continue
		}
		out.plusInside[i] = plus
		out.takeInterior(up, down, plus)
	}
	return out
}

// takeInterior records the FIRST point the probe found strictly inside the region the shell bounds —
// the point shellEnclosedBy answers at. Keeping the first makes the answer depend only on the shell's
// member order, which is sorted (walkFaceComponents).
func (s *shellSides) takeInterior(up, down math.Point3, plusInside bool) {
	if s.found {
		return
	}
	s.interior, s.found = down, true
	if plusInside {
		s.interior = up
	}
}

// faceSideInShell steps a stand-off either side of one face and reports which side lies inside the
// region the shell bounds, with both stepped points. ok=false when the two sides agree or the casts
// grazed, which reads nothing.
func faceSideInShell(own *shellProbe, f *fluxFace, step float64) (up, down math.Point3, plusInside, ok bool) {
	q, n, okP := faceProbePoint(f)
	if !okP {
		return up, down, false, false
	}
	up, down = q.TranslateBy(n.Scale(math.Scalar(step))), q.TranslateBy(n.Scale(math.Scalar(-step)))
	plus, okUp := own.parityInside(up)
	minus, okDown := own.parityInside(down)
	if !okUp || !okDown || plus == minus {
		return up, down, false, false
	}
	return up, down, plus, true
}

// shellProbedSigns turns the orientation-free side readings into outward signs, once the shell's ROLE
// is known: the material of a lump is inside the region it bounds and that of a void outside it, so the
// sign is +1 exactly when the +S_u×S_v side is the non-material one. These are the VOTES shellBit
// counts — never a per-face verdict; a face's own sign comes from its loops (see orientFaceSigns). A
// face the probe could not read carries no entry and abstains.
func shellProbedSigns(sides shellSides, enclosed bool) map[int]float64 {
	out := make(map[int]float64, len(sides.plusInside))
	for i, plus := range sides.plusInside {
		out[i] = -1
		if plus == enclosed {
			out[i] = 1
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
// lies within theirs and a point of the region it bounds classifies inside them. The box gate is what
// keeps two lumps that TOUCH — a rod's stubs either side of a near-pinch cut, whose boundaries pass
// within the tolerance of each other — from being classified at a point the flux cannot read.
//
// The point is one taken strictly INSIDE the region the shell bounds (shellSides.interior), never a
// point of its boundary. It used to be a loop VERTEX, and a vertex is exactly where the question has no
// answer: it lies on the shell being classified AND, where two lumps kiss, on the shells it is being
// classified against, so the nearest crossing along every ray is the self-hit at t≈0 and the side read
// from it is a coin flip. A cut that severed a lump touching the rest — a slot between two already-cut
// slots of the Inventor multipoint disk, and the 22-face plate fixture that reproduces it — read the
// severed lump as a void and inverted all of its faces' stored senses against their own loop winding
// (Oblikovati/Oblikovati#3512).
func shellEnclosedBy(oriented, faces []fluxFace, shell []int, sides shellSides) bool {
	if len(oriented) == 0 || !sides.found {
		return false
	}
	box := paddedBox(fluxFacesBox(oriented), probeOffsetRel)
	if !box.ContainsBox(fluxFacesBox(fluxFacesAt(faces, shell))) {
		return false
	}
	q := &fluxQuery{faces: oriented}
	return q.inside(sides.interior, box)
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
	// trims holds each face's boundary DEVELOPED into (u, v), also measured once, for the same reason
	// and at the same cost: the ray traversal asks each face "is this pierce inside your trim" per ray,
	// and developing it there re-projected the whole boundary every time (ADR-0063).
	trims []*faceTrimUV
	box   math.Box
}

// newShellProbe boxes every face of the shell, each padded by a share of its own diagonal so the
// sampled box can never miss a curved face's true extent between samples (a 45° sample step on a
// rim leaves a 7.6% sagitta; the pad is 10%).
func newShellProbe(faces []fluxFace) *shellProbe {
	p := &shellProbe{faces: curvedFacesOf(faces), boxes: make([]math.Box, len(faces)),
		bands: make([]float64, len(faces)), trims: make([]*faceTrimUV, len(faces)), box: fluxFacesBox(faces)}
	for i := range faces {
		p.boxes[i] = paddedBox(fluxFacesBox(faces[i:i+1]), probeBoxPadRel)
		p.bands[i] = faceBoundaryBand(p.faces[i])
		p.trims[i] = developFaceTrim(p.faces[i])
		p.trims[i].index = newChartIndex(p.trims[i].chartContours())
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
			n, clean := faceRayCrossingsDeveloped(f, s.trims[i], ray, tMax, s.bands[i], tol)
			if !clean {
				return false, false
			}
			crossings += n
		}
		return crossings%2 == 1, true
	})
}
