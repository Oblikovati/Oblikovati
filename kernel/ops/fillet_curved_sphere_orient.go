// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// orientForSphereHost fixes the weld's global loop sense for the sphere-patch mesher when a sphere HOST
// face is present (D5/E4 sub-hemisphere, D9 >hemisphere). The mesher (spherePatchMesh/interiorAxis)
// resolves which of the two regions a sphere loop bounds from the loop's ABSOLUTE winding, but the weld's
// global sense is otherwise free: orientFilletShell (in assembleBody) only unifies RELATIVE windings and
// pins the ABSOLUTE sense to its SEED (faces[0]) — an arm face — so the host sphere inherits an arbitrary
// sense and can mesh the COMPLEMENT (D5: 224929 vs ~57815; D9: 103387 vs ~179292). The fix seeds the shell
// from the host sphere itself, wound so the mesher fills the MATERIAL zone: move that face to faces[0] (so
// orientFilletShell keeps its sense) and reverse it first when it would otherwise fill the complement. It
// reads the material-zone pole off the ORIGINAL face's sampled loop (originalZonePole) — the four bare
// bite vertices alone cannot place it (both regions share them). It reseeds EXACTLY the hosts base did:
// only when the bite loop is COMPACT (compactSpherePole — its bare verts fit in a hemisphere: true for the
// D5 sub-hemisphere cap AND the D9 reflex corner, whose SMALL bite loop is compact even though its material
// zone is >hemisphere; false for the E4 wide crescent, which base left to the mesher's own winding — the
// D9-T2 review's byte-identity gate). Fires ONLY for a genuine host sphere (from curvedHostFaces — never
// the convex corner-blend patch `sf`, not in hostFaces); non-sphere-host welds (the rest of the corpus)
// are untouched.
func orientForSphereHost(body *topo.Body, all, hostFaces []filletFace) []filletFace {
	if body == nil {
		return all // no original body to anchor the material interior — do-no-harm (never in the weld path)
	}
	i, ok := hostSphereIndex(all, hostFaces)
	if !ok {
		return all
	}
	sph := all[i].surface.(geom.Sphere)
	if _, compact := compactSpherePole(sph, all[i].loops[0].pts); !compact {
		return all // non-compact BITE loop (E4): base left it to the mesher's own winding — do not reseed (D9-T2 review)
	}
	// The corner-path winding test reads the RAW bite verts (byte-identity with the pre-D9 base): the
	// compact bite loop's chords do not shortcut across the material zone the way a wide loop's would.
	return seedSphereHostSense(body, all, i, all[i].loops[0].pts)
}

// seedSphereHostSense reorders `all` so the host sphere at index i becomes the shell seed (faces[0]) wound
// so the sphere-patch mesher fills the MATERIAL zone. windRing is the loop sampled for the winding-sign test:
// the raw compact bite verts for the corner path, or the arc-sampled boundary for a WIDE single-runout loop
// (whose 270° rail/rim chords would otherwise shortcut across the zone and misread the sign). The material
// pole comes from originalZonePole (the original outward face), correct for a sub- and a >hemisphere zone.
func seedSphereHostSense(body *topo.Body, all []filletFace, i int, windRing []math.Point3) []filletFace {
	sph := all[i].surface.(geom.Sphere)
	pole, ok := originalZonePole(body, sph)
	if !ok {
		return all
	}
	if loopTurnsNegative(sph, pole, windRing) {
		all[i] = reverseFilletFace(all[i]) // emit it CCW-seen-from-outside so the seeded sense meshes the patch
	}
	all[0], all[i] = all[i], all[0] // seed orientFilletShell from the host sphere (it keeps faces[0]'s sense)
	return all
}

// filletLoopWindRing samples a pre-assembly filletLoop into a closed polyline (each Arc3d segment expanded
// through its interior points via segPolyline) for the sphere-host winding-sign test. A wide single-runout
// sphere loop is mostly reflex arcs (the 270° rail and rim), so its raw vertices alone chord across the
// removed zone; sampling the arcs makes loopTurnsNegative read the true material-zone winding.
func filletLoopWindRing(l filletLoop) []math.Point3 {
	n := len(l.pts)
	segs := make([]endSeg, n)
	for i := range l.pts {
		s := endSeg{from: l.pts[i], to: l.pts[(i+1)%n]}
		if arc, ok := curveAt(l.curves, i).(geom.Arc3d); ok {
			s.curve, s.mid, s.arc = arc, arc.PointAt(0.5), true
		}
		segs[i] = s
	}
	return segPolyline(segs)
}

// cornerBlendMeshesComplement reports whether the assembled body's CORNER BLEND sphere patch would mesh
// the COMPLEMENT rather than the sub-hemisphere cap it should. The sphere-patch mesher reads the patch
// loop's ABSOLUTE winding to pick which of the two regions it bounds, but orientFilletShell pins the
// shell's absolute sense to an arbitrary arm seed — so the corner cap can land inverted (D1: 1016.7 vs the
// exact 238.5 spherical triangle). The corner blend is always the COMPACT sub-hemisphere triangle around
// its tangent-points' mean direction, so it meshes the complement exactly when its (sampled) loop turns
// NEGATIVE about that mean pole. Fires ONLY when the ORIGINAL body carries no host sphere (the cylinder-
// host B3 and cone-host C2/C6/D1 corners): a sphere-host corner (D5/E4/D9) has TWO sphere faces and its
// winding is owned by orientForSphereHost, so a whole-shell flip there could invert the host too — leave it.
func cornerBlendMeshesComplement(originalBody, built *topo.Body) bool {
	if hasSphereFace(originalBody) {
		return false // sphere-host corner: orientForSphereHost owns the sphere winding — never uniform-flip
	}
	f, sph, ok := cornerBlendSphereFace(built)
	if !ok {
		return false // no sphere corner blend (a canal/coons4 patch) — its mesher is winding-independent
	}
	// The corner blend is ALWAYS the sub-hemisphere spherical triangle (Girard area ≤ ~448 for these
	// corners, well under a hemisphere 2πr²). Reading the sphere-patch mesher's region directly — tessellate
	// and compare to the hemisphere area — is robust where a loop-winding heuristic is not: a THIN wide
	// triangle (C2's ~90° arcs, 206.9 area) confuses a single-pole winding test but its MESHED area cleanly
	// separates the cap (< 2πr²) from the complement (> 2πr², D1's 1016.7). No false positive on a correct
	// corner ⇒ the uniform flip never fires on B3 or the 60 greens (byte-identity holds).
	area := meshGeometryProperties(TessellateFace(f, PropertyQuality())).Area
	return area > 2*stdmath.Pi*sph.Radius*sph.Radius
}

// cornerBlendSphereFace returns the assembled body's single sphere face — the corner blend patch of a
// cone/cylinder-host corner (its only sphere). ok=false when no sphere face is present.
func cornerBlendSphereFace(b *topo.Body) (*topo.Face, geom.Sphere, bool) {
	for _, f := range b.Faces() {
		if s, ok := f.Geometry().(geom.Sphere); ok {
			return f, s, true
		}
	}
	return nil, geom.Sphere{}, false
}

// hasSphereFace reports whether the body carries any spherical face — the marker of a sphere-HOST corner
// (whose corner-blend winding is owned by orientForSphereHost, not the uniform-flip fixup).
func hasSphereFace(b *topo.Body) bool {
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Sphere); ok {
			return true
		}
	}
	return false
}

// hostSphereIndex returns the index in `all` of a host sphere face (one carrying a boundary loop), or
// ok=false when none is present. The host sphere is matched by surface identity to a curvedHostFaces entry.
func hostSphereIndex(all, hostFaces []filletFace) (int, bool) {
	for _, hf := range hostFaces {
		if _, isSphere := hf.surface.(geom.Sphere); !isSphere || len(hf.loops) == 0 || len(hf.loops[0].pts) < 3 {
			continue
		}
		for i := range all {
			if all[i].surface == hf.surface && len(all[i].loops) > 0 {
				return i, true
			}
		}
	}
	return 0, false
}

// originalZonePole returns a direction inside the host sphere's MATERIAL zone, taken from the ORIGINAL
// imported sphere face of the same surface. The original face is a valid solid boundary, so its loop is
// stored in a face-consistent (outward, CCW-seen-from-outside) traversal — the Newell area vector of that
// loop (∝ the region's spherical vector area ∫∫ r̂ dA) therefore points at the MATERIAL zone's centroid for
// BOTH a sub-hemisphere (D5/E4) and a >hemisphere (D9) host, where the boundary mean would point at the
// removed region instead. The loop's arc edges are sampled (segPolyline) so the Newell integral resolves.
// ok=false when no original sphere face matches or the loop degenerates.
func originalZonePole(body *topo.Body, sph geom.Sphere) (math.Vector3, bool) {
	for _, f := range body.Faces() {
		if s, ok := f.Geometry().(geom.Sphere); !ok || s != sph {
			continue
		}
		loop := outerHostLoop(f)
		if loop == nil {
			return math.Vector3{}, false
		}
		ring := segPolyline(segsFromLoop(loop))
		if len(ring) < 3 {
			return math.Vector3{}, false
		}
		pole, err := math.UnitVector3FromVector(sphereLoopVectorArea(sph, ring))
		if err != nil {
			return math.Vector3{}, false
		}
		return pole.AsVector(), true
	}
	return math.Vector3{}, false
}

// sphereLoopVectorArea is the discrete spherical vector area of a boundary ring on sphere sph: (1/2) Σ
// (p_i−c) × (p_{i+1}−c), which points toward the centroid of the region the ring bounds CCW-seen-from-
// outside (its ∫∫ r̂ dA). Used to read the material zone's interior from the original face's outward loop.
func sphereLoopVectorArea(sph geom.Sphere, ring []math.Point3) math.Vector3 {
	var sum math.Vector3
	for i := range ring {
		a := sph.Center.VectorTo(ring[i])
		b := sph.Center.VectorTo(ring[(i+1)%len(ring)])
		sum = sum.Add(a.Cross(b))
	}
	return sum
}

// compactSpherePole returns the bite loop's mean outward direction as a compact pole, ok only when EVERY
// boundary vertex lies within 90° of it (the bite is a genuinely compact patch — its corner rim fits in a
// hemisphere). This is the seed GATE, reproducing base's decision set EXACTLY: the D5 cap and the D9 reflex
// corner (both compact bite loops) are reseeded; the E4 wide-crescent bite (a vertex >90° from the mean) is
// NOT, so E4 stays byte-identical to base (D9-T2 review). The returned pole is only the gate cue; the
// reversal pole comes from originalZonePole (the material zone), which is correct for a >hemisphere host too.
func compactSpherePole(sph geom.Sphere, pts []math.Point3) (math.Vector3, bool) {
	var sum math.Vector3
	for _, p := range pts {
		sum = sum.Add(sphereDir(sph, p))
	}
	mean, err := math.UnitVector3FromVector(sum)
	if err != nil {
		return math.Vector3{}, false
	}
	pole := mean.AsVector()
	for _, p := range pts {
		if float64(sphereDir(sph, p).Dot(pole)) <= 0 {
			return math.Vector3{}, false // a vertex >90° from the mean — a wide (E4-like) bite, not compact
		}
	}
	return pole, true
}

// loopTurnsNegative reports whether the loop turns clockwise (NOT CCW-seen-from-outside) around the compact
// pole — the winding sense the sphere-patch mesher's sign test misreads as the complement.
func loopTurnsNegative(sph geom.Sphere, pole math.Vector3, pts []math.Point3) bool {
	center := sph.Center.TranslateBy(pole.Scale(math.Scalar(sph.Radius)))
	return loopWindingAround(pts, center, pole) < 0
}

// reverseFilletFace reverses every loop of a face (metadata-preserving), so orientForSphereHost can wind
// the seed host sphere CCW-seen-from-outside — via the same reverseFilletLoop the shell 2-colouring uses.
func reverseFilletFace(f filletFace) filletFace {
	loops := make([]filletLoop, len(f.loops))
	for i, l := range f.loops {
		loops[i] = reverseFilletLoop(l)
	}
	return filletFace{surface: f.surface, loops: loops, parent: f.parent}
}
