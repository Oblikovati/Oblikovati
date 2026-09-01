// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The O1 class's own guards: the role predicate, the three-way CLASS-DISJOINTNESS matrix (design step S4 —
// each builder must decline every other class's role signature, so the ordered ladder's verdict never depends
// on order), and the shared-executor mid-rail orientation fix slice 2 needed.

// o1RoleArms is O1's role signature as bare edgeFillets: a CONCAVE cylinder arm, a CONCAVE cove torus arm,
// and a CONVEX Plane∧Plane band. Only the Go types and the flip/concave flags matter to the predicates, so
// the placements are the fixed frame the other role tests use.
func o1RoleArms(t *testing.T) []edgeFillet {
	t.Helper()
	pl := rolePlane(t)
	return []edgeFillet{
		{armSurface: roleCyl(t), armConcave: true},
		{armSurface: roleTorus(t), armConcave: true},
		{armSurface: roleCyl(t), a: minimalRoleFace(pl, 30), b: minimalRoleFace(pl, 31)},
	}
}

// m8RoleArms is M8's role signature: a CONVEX Cyl∧Plane pivot, a concave cove torus, and a CONCAVE
// (flipped) planar band.
func m8RoleArms(t *testing.T) []edgeFillet {
	t.Helper()
	pl := rolePlane(t)
	return []edgeFillet{
		{armSurface: roleCyl(t), a: minimalRoleFace(pl, 32), b: minimalRoleFace(roleCyl(t), 33)},
		{armSurface: roleTorus(t), armConcave: true},
		{flip: true, a: minimalRoleFace(pl, 34), b: minimalRoleFace(pl, 35)},
	}
}

// n4RoleArms is N4's role signature: a concave cylinder arm, a CONVEX torus arm, and a CONCAVE planar band.
func n4RoleArms(t *testing.T) []edgeFillet {
	t.Helper()
	pl := rolePlane(t)
	return []edgeFillet{
		{armSurface: roleCyl(t), armConcave: true},
		{armSurface: roleTorus(t)},
		{flip: true, a: minimalRoleFace(pl, 36), b: minimalRoleFace(pl, 37)},
	}
}

// TestIsConvexBandArmDeclines pins the O1 lateral-arm predicate: only a CONVEX, non-flipped Cylinder arm
// whose BOTH hosts are planes qualifies. The concave cyl arm, the flipped (concave) band and a Plane∧Cylinder
// convex pivot — M8's arm, whose second host is a cylinder — all decline.
func TestIsConvexBandArmDeclines(t *testing.T) {
	t.Parallel()
	pl := rolePlane(t)
	if isConvexBandArm(edgeFillet{armSurface: roleCyl(t), armConcave: true, a: minimalRoleFace(pl, 40), b: minimalRoleFace(pl, 41)}) {
		t.Error("isConvexBandArm accepted a CONCAVE cylinder arm — the O1 band is convex")
	}
	if isConvexBandArm(edgeFillet{armSurface: roleCyl(t), flip: true, a: minimalRoleFace(pl, 42), b: minimalRoleFace(pl, 43)}) {
		t.Error("isConvexBandArm accepted a FLIPPED (concave) planar band")
	}
	if isConvexBandArm(edgeFillet{armSurface: roleCyl(t), a: minimalRoleFace(pl, 44), b: minimalRoleFace(roleCyl(t), 45)}) {
		t.Error("isConvexBandArm accepted a Plane∧Cylinder convex pivot — M8's arm, not O1's band")
	}
	if !isConvexBandArm(edgeFillet{armSurface: roleCyl(t), a: minimalRoleFace(pl, 46), b: minimalRoleFace(pl, 47)}) {
		t.Error("isConvexBandArm rejected a convex Plane∧Plane band — O1's lateral arm")
	}
}

// TestMixedCornerClassesAreDisjoint is the design's S4 obligation, as a matrix: each of the three mixed
// trihedral classifiers must accept ONLY its own role signature. Order in cornerPlanBuilders() then cannot
// change any verdict, and adding a class cannot silently steal another's cases.
func TestMixedCornerClassesAreDisjoint(t *testing.T) {
	t.Parallel()
	signatures := map[string][]edgeFillet{"M8": m8RoleArms(t), "N4": n4RoleArms(t), "O1": o1RoleArms(t)}
	accepts := map[string]func([]edgeFillet) bool{
		"M8": func(a []edgeFillet) bool { _, ok := classifyMixedRoleArms(a); return ok },
		"N4": func(a []edgeFillet) bool { _, ok := classifyN4MixedArms(a); return ok },
		"O1": func(a []edgeFillet) bool { _, ok := classifyO1MixedArms(a); return ok },
	}
	for classifier, accept := range accepts {
		for signature, arms := range signatures {
			got, want := accept(arms), classifier == signature
			if got != want {
				t.Errorf("classify%s(%s roles) = %v, want %v — the mixed corner classes must be disjoint",
					classifier, signature, got, want)
			}
		}
	}
}

// TestClassifyO1DeclinesDupRoles pins the dup-role guard: two arms of the same role is never the O1 1+1+1
// config, and a two-arm or four-arm corner is a different class entirely.
func TestClassifyO1DeclinesDupRoles(t *testing.T) {
	t.Parallel()
	arms := o1RoleArms(t)
	if _, ok := classifyO1MixedArms([]edgeFillet{arms[0], arms[0], arms[2]}); ok {
		t.Error("classifyO1MixedArms accepted TWO concave cylinder arms")
	}
	if _, ok := classifyO1MixedArms(arms[:2]); ok {
		t.Error("classifyO1MixedArms accepted a 2-arm corner")
	}
	if _, ok := classifyO1MixedArms(append(append([]edgeFillet{}, arms...), arms[1])); ok {
		t.Error("classifyO1MixedArms accepted a 4-arm corner")
	}
}

// TestOrientMidToChainIsGeometricNotPositional is the regression test for the shared-executor defect slice 2
// found (cornerweld_splice.go): the patch's on-host mid rail is registered ONCE in the patch ring's D→A
// direction, so which end continues the first bite's chain depends on which arm the class builder happens to
// list first — N4's ring starts at its first arm's foot, O1's at its second's. Deciding it positionally
// produced a loop whose segments were shifted by one: Valid ∧ Manifold, with four 1-incident edges and an
// unclosed shell. Both orientations must now resolve, and a chain reaching NEITHER foot must decline.
func TestOrientMidToChainIsGeometricNotPositional(t *testing.T) {
	t.Parallel()
	d, a := math.P3(1, 0, 0), math.P3(0, 1, 0)
	mid := []endSeg{{from: d, to: a, curve: geom.NewLineSegment(d, a), mid: d.Midpoint(a)}}
	forward, ok := orientMidToChain(mid, d, 1e-9)
	if !ok || float64(forward[0].from.DistanceTo(d)) > 1e-9 {
		t.Fatalf("orientMidToChain(head=D) = %v (ok=%v), want the chain as registered", forward, ok)
	}
	reversed, ok := orientMidToChain(mid, a, 1e-9)
	if !ok || float64(reversed[0].from.DistanceTo(a)) > 1e-9 {
		t.Fatalf("orientMidToChain(head=A) = %v (ok=%v), want the chain reversed", reversed, ok)
	}
	if _, ok := orientMidToChain(mid, math.P3(5, 5, 5), 1e-9); ok {
		t.Error("orientMidToChain accepted a head the chain does not reach — an inconsistent plan must floor")
	}
	if got, ok := orientMidToChain(nil, d, 1e-9); !ok || got != nil {
		t.Errorf("orientMidToChain(nil) = %v (ok=%v), want (nil, true) — a triple point has nothing to orient", got, ok)
	}
}
