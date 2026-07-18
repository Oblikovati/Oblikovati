// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// orientForSphereHost fixes the weld's global loop sense for the sphere-patch mesher when a sub-hemisphere
// sphere HOST face is present (D5/E4). The mesher (spherePatchMesh/interiorAxis) resolves which of the two
// regions a sphere loop bounds from the loop's ABSOLUTE winding, but the weld's global sense is otherwise
// free: orientFilletShell (in assembleBody) only unifies RELATIVE windings and pins the ABSOLUTE sense to
// its SEED (faces[0]) — an arm face — so the host sphere inherits an arbitrary sense and can mesh the
// COMPLEMENT (D5: 224929 vs ~57815). The fix seeds the shell from the host sphere itself, wound
// CCW-seen-from-outside around its compact pole: move that face to faces[0] (so orientFilletShell keeps its
// sense) and reverse it first when it came out clockwise. It fires ONLY for a genuine host sphere (from
// curvedHostFaces — never the convex corner-blend patch, which is `sf`, not in hostFaces) that is
// sub-hemispheric, so a >hemisphere host (D6) and every non-sphere-host weld (the whole existing corpus)
// are untouched. Sub-hemisphere host spheres only arise on the curved-ARM weld (weldCurvedArmFaces, its caller).
func orientForSphereHost(all, hostFaces []filletFace) []filletFace {
	i, pole, ok := subHemisphereSphereHost(all, hostFaces)
	if !ok {
		return all
	}
	if loopTurnsNegative(all[i].surface.(geom.Sphere), pole, all[i].loops[0].pts) {
		all[i] = reverseFilletFace(all[i]) // emit it CCW-seen-from-outside so the seeded sense meshes the patch
	}
	all[0], all[i] = all[i], all[0] // seed orientFilletShell from the host sphere (it keeps faces[0]'s sense)
	return all
}

// subHemisphereSphereHost finds a host sphere face whose whole boundary lies within a hemisphere (a compact
// pole exists), returning its index in `all` and that pole. ok=false when no host face is a sub-hemisphere
// sphere (a >hemisphere host — D6 — has no compact pole and is left to the mesher's winding test).
func subHemisphereSphereHost(all, hostFaces []filletFace) (int, math.Vector3, bool) {
	for _, hf := range hostFaces {
		sph, isSphere := hf.surface.(geom.Sphere)
		if !isSphere || len(hf.loops) == 0 || len(hf.loops[0].pts) < 3 {
			continue
		}
		pole, compact := compactSpherePole(sph, hf.loops[0].pts)
		if !compact {
			continue
		}
		for i := range all {
			if all[i].surface == hf.surface && len(all[i].loops) > 0 {
				return i, pole, true
			}
		}
	}
	return 0, math.Vector3{}, false
}

// compactSpherePole returns the loop's mean outward direction as the patch's compact pole, ok only when
// EVERY boundary vertex lies within 90° of it (a genuinely sub-hemisphere patch, so the pole is
// unambiguously inside the patch). ok=false for a >hemisphere host (D6), where the winding is the only cue.
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
			return math.Vector3{}, false // a vertex >90° from the mean — not a sub-hemisphere patch
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
