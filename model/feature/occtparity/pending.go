// SPDX-License-Identifier: GPL-2.0-only

package occtparity

// pendingCapability is the corpus-level PENDING list: cases whose fillet capability is NOT YET
// BUILT. Each entry carries the engine's own measured decline reason, so the list reads as a
// specification of the remaining work rather than as suppression. It exists so the per-case parity
// gate (model/feature's TestOCCTBlend* grids) reports the state of the engine instead of blocking
// the pipeline on 104 known, documented gaps — the fillet effort is SUSPENDED, not abandoned; see
// architecture/audits/fillet-occt-parity-audit-2026-08.md (Appendix A enumerates these by lane).
//
// IT IS A HARD RATCHET IN BOTH DIRECTIONS, and that is what separates it from suppression:
//   - a listed case that starts PASSING fails the gate, demanding its entry be deleted, so the list
//     can never hide progress;
//   - an unlisted case that fails still fails the gate, so it can never hide a regression;
//   - pendingCapabilityCount pins the size, so growing the list is a deliberate, reviewed act.
//
// It is DISTINCT from quarantine.go, which means something narrower: "this must not count as green
// because a passing area would be COINCIDENTAL, masking broken geometry." Quarantine is empty and
// must stay empty (TestTheCorpusHoldsNoCase). Never move a case here to make an area pass, and never
// move one to quarantine merely to silence it.
//
// An entry leaves ONLY when the capability is built and the case genuinely greens — never by
// widening a tolerance.
var pendingCapability = map[quarantineKey]string{
	{"bfuseblend", "A8"}:       "edge bounds 1 faces, need 2",
	{"bfuseblend", "A9"}:       "curved (torus-arm) Plane∧Cylinder edge 704 could not be welded into the solid: assembled weld did not certify as a valid solid",
	{"bfuseblend", "B2"}:       "cannot round an edge bordering a curved (geom.BSplineSurface) face — rounding a filleted or otherwise curved edge is not yet supported",
	{"bfuseblend", "B6"}:       "curved miter arms unsupported at vertex 1333 (need one torus + one cylinder equal-r arm, or two coaxial tori; radius 10)",
	{"bfuseblend", "B7"}:       "curved miter arms unsupported at vertex 1374 (need one torus + one cylinder equal-r arm, or two coaxial tori; radius 10)",
	{"complex", "B9"}:          "edge endpoint has no end face to round",
	{"complex", "C1"}:          "rounding an edge that borders a curved (cylinder) face is not yet supported",
	{"complex", "C2"}:          "mixed-radius corner where 6 filleted edges meet a 6-face vertex is not a supported blend (need 2 edges sharing a face, or 3 edges at a trihedral vertex)",
	{"complex", "C5"}:          "curved miter arms unsupported at vertex 1638 (need one torus + one cylinder equal-r arm, or two coaxial tori; radius 10)",
	{"complex", "C6"}:          "curved (cylinder-arm) Plane∧Cylinder edge 1697 could not be welded into the solid: curved arms do not meet at one shared trihedral vertex",
	{"complex", "D2"}:          "mixed-radius trihedral corner (3 faces, radii [10 5 5]) needs a torus corner patch — not yet supported",
	{"complex", "D6"}:          "rim radius 59 must be in (0, cylinder radius 20)",
	{"complex", "D8"}:          "area +3.577% outside OCCT tolerance — geometry still wrong; must NOT be counted green",
	{"complex", "E3"}:          "mixed-radius trihedral corner (3 faces, radii [14 7 7]) needs a torus corner patch — not yet supported",
	{"complex", "E5"}:          "curved (torus-arm) Plane∧Cylinder edge 2083 could not be welded into the solid: curved arms do not meet at one shared trihedral vertex",
	{"complex", "E7"}:          "edge between a cylinder and a tangent plane is smooth (no corner to round)",
	{"complex", "E8"}:          "cannot round an edge bordering a curved (cylinder) face — rounding a filleted or otherwise curved edge is not yet supported",
	{"complex", "F1"}:          "rounding an edge that borders a curved (cylinder) face is not yet supported",
	{"complex", "F2"}:          "area +4.682% outside OCCT tolerance — geometry still wrong; must NOT be counted green",
	{"complex", "F4"}:          "curved miter seam did not close at vertex 2454 (radius 10)",
	{"complex", "F7"}:          "curved (torus-arm) Plane∧Cylinder edge 2479 could not be welded into the solid: curved arms do not meet at one shared trihedral vertex",
	{"complex", "G1"}:          "torus∧torus miter seam did not close at vertex 2517 (radius 3)",
	{"complex", "G2"}:          "curved (torus-arm) Plane∧Cylinder edge 2589 could not be welded into the solid: trihedral corner needs 3 arms (got 2 at vertex 2588)",
	{"encoderegularity", "A1"}: "curved miter arms unsupported at vertex 2629 (need one torus + one cylinder equal-r arm, or two coaxial tori; radius 3)",
	{"encoderegularity", "A4"}: "corner face must be planar",
	{"simple", "E5"}:           "curved (geom.BSplineSurface-arm) Plane∧Cylinder edge 5941 could not be welded into the solid: trihedral corner needs 3 arms (got 1 at vertex 5940)",
	{"simple", "E6"}:           "curved (geom.BSplineSurface-arm) Plane∧Cylinder edge 5969 could not be welded into the solid: corner solve declined (station gap / host non-tangency / closure failure)",
	{"simple", "E8"}:           "curved (torus-arm) Plane∧Cylinder edge 6058 could not be welded into the solid: corner solve declined (station gap / host non-tangency / closure failure)",
	{"simple", "E9"}:           "curved (geom.BSplineSurface-arm) Plane∧Cylinder edge 6088 could not be welded into the solid: trihedral corner needs 3 arms (got 1 at vertex 6087)",
	{"simple", "F1"}:           "curved (geom.BSplineSurface-arm) Plane∧Cylinder edge 6116 could not be welded into the solid: corner solve declined (station gap / host non-tangency / closure failure)",
	{"simple", "F3"}:           "curved (geom.BSplineSurface-arm) Plane∧Cylinder edge 6207 could not be welded into the solid: corner solve declined (station gap / host non-tangency / closure failure)",
	{"simple", "F5"}:           "corner face must be planar",
	{"simple", "G3"}:           "cannot round edge 6481 — it runs into an existing rounded (geom.BSplineSurface) face at its end; fillet these edges BEFORE the adjacent rounds, or select them together",
	{"simple", "G4"}:           "curved miter arms unsupported at vertex 6547 (need one torus + one cylinder equal-r arm, or two coaxial tori; radius 10)",
	{"simple", "G6"}:           "corner face must be planar",
	{"simple", "G8"}:           "curved miter arms unsupported at vertex 6687 (need one torus + one cylinder equal-r arm, or two coaxial tori; radius 5)",
	{"simple", "H1"}:           "curved miter arms unsupported at vertex 6757 (need one torus + one cylinder equal-r arm, or two coaxial tori; radius 5)",
	{"simple", "H2"}:           "cannot round an edge bordering a curved (cone) face — rounding a filleted or otherwise curved edge is not yet supported",
	{"simple", "H3"}:           "corner where 3 filleted edges meet a 2-face vertex is not a supported blend (need 3 edges at a trihedral vertex, or 2 edges sharing a face)",
	{"simple", "H4"}:           "edge endpoint has no end face to round",
	{"simple", "H5"}:           "corner where 3 filleted edges meet a 2-face vertex is not a supported blend (need 3 edges at a trihedral vertex, or 2 edges sharing a face)",
	{"simple", "H7"}:           "curved (torus-arm) Plane∧Cylinder edge 6990 could not be welded into the solid: corner solve declined (station gap / host non-tangency / closure failure)",
	{"simple", "H9"}:           "curved (geom.BSplineSurface-arm) Plane∧Cylinder edge 7016 could not be welded into the solid: arm rail bundle declined (geom.Cylinder): oblique runout: a host spring does not cross the capping face (spring∩cap foot declined)",
	{"simple", "I2"}:           "curved miter arms unsupported at vertex 7119 (need one torus + one cylinder equal-r arm, or two coaxial tori; radius 10)",
	{"simple", "I4"}:           "curved (geom.BSplineSurface-arm) Plane∧Cylinder edge 7230 could not be welded into the solid: corner solve declined (station gap / host non-tangency / closure failure)",
	{"simple", "I6"}:           "curved (ops.bsplineHostArmSurface-arm) Plane∧Cylinder edge 7326 could not be welded into the solid: curved arms do not meet at one shared trihedral vertex",
	{"simple", "I8"}:           "curved miter arms unsupported at vertex 7437 (need one torus + one cylinder equal-r arm, or two coaxial tori; radius 10)",
	{"simple", "J9"}:           "cannot round an edge bordering a curved (geom.BSplineSurface) face — rounding a filleted or otherwise curved edge is not yet supported",
	{"simple", "L8"}:           "curved (torus-arm) Plane∧Cylinder edge 11018 could not be welded into the solid: corner solve declined (station gap / host non-tangency / closure failure)",
	{"simple", "M3"}:           "curved (torus-arm) Plane∧Cylinder edge 11619 could not be welded into the solid: trihedral corner needs 3 arms (got 2 at vertex 11618)",
	{"simple", "M6"}:           "curved miter arms unsupported at vertex 11938 (need one torus + one cylinder equal-r arm, or two coaxial tori; radius 5)",
	{"simple", "M9"}:           "curved (torus-arm) Plane∧Cylinder edge 12258 could not be welded into the solid: trihedral corner needs 3 arms (got 2 at vertex 12256)",
	{"simple", "N2"}:           "curved (torus-arm) Plane∧Cylinder edge 12429 could not be welded into the solid: curved arms do not meet at one shared trihedral vertex",
	{"simple", "N8"}:           "curved (torus-arm) Plane∧Cylinder edge 13202 could not be welded into the solid: curved arms do not meet at one shared trihedral vertex",
	{"simple", "O2"}:           "curved (cylinder-arm) Plane∧Cylinder edge 13453 could not be welded into the solid: curved arms do not meet at one shared trihedral vertex",
	{"simple", "O3"}:           "cannot round an edge bordering a curved (cylinder) face — rounding a filleted or otherwise curved edge is not yet supported",
	{"simple", "O4"}:           "corner face must be planar",
	{"simple", "O5"}:           "curved miter arms unsupported at vertex 13573 (need one torus + one cylinder equal-r arm, or two coaxial tori; radius 5)",
	{"simple", "O6"}:           "cannot round an edge bordering a curved (cylinder) face — rounding a filleted or otherwise curved edge is not yet supported",
	{"simple", "O7"}:           "curved miter arms unsupported at vertex 13651 (need one torus + one cylinder equal-r arm, or two coaxial tori; radius 5)",
	{"simple", "O9"}:           "curved (cylinder-arm) Plane∧Cylinder edge 13737 could not be welded into the solid: arm rail bundle declined (geom.Torus): far vertex 13730: a second filleted edge 13740 also ends here (fillet-fillet interference / setback regime, out of scope)",
	{"simple", "P2"}:           "curved miter seam did not close at vertex 13815 (radius 5)",
	{"simple", "P3"}:           "curved miter seam did not close at vertex 13843 (radius 5)",
	{"simple", "P4"}:           "curved (cylinder-arm) Plane∧Cylinder edge 13882 could not be welded into the solid: single-arm runout: host geom.Cylinder retrim declined (bite {48.14814814814814 0.03430532136280817 60}→{48.14814814814814 0.03430532136280817 150})",
	{"simple", "P5"}:           "curved (cylinder-arm) Plane∧Cylinder edge 13910 could not be welded into the solid: curved miter: shared host geom.Cylinder's own boundary is degenerate (pre-existing defect, not a fillet gap)",
	{"simple", "P6"}:           "cannot round an edge bordering a curved (cylinder) face — rounding a filleted or otherwise curved edge is not yet supported",
	{"simple", "P7"}:           "cannot round an edge bordering a curved (cylinder) face — rounding a filleted or otherwise curved edge is not yet supported",
	{"simple", "Q2"}:           "curved (torus-arm) Plane∧Cylinder edge 14289 could not be welded into the solid: trihedral corner needs 3 arms (got 1 at vertex 14287)",
	{"simple", "Q6"}:           "rim radius 2000 must be in (0, cylinder radius 1000)",
	{"simple", "R7"}:           "rounding an edge that borders a curved (cylinder) face is not yet supported",
	{"simple", "R8"}:           "area +17.324% outside OCCT tolerance — geometry still wrong; must NOT be counted green",
	{"simple", "S2"}:           "curved (torus-arm) Plane∧Cylinder edge 16473 could not be welded into the solid: concave closed-rim band: plane-contact radius 16.2462 exceeds the cap face half-extent 15 — the cove spills off the plate onto the adjacent walls (a multi-face concave interaction is a follow-on slice)",
	{"simple", "S5"}:           "curved (torus-arm) Plane∧Cylinder edge 18898 could not be welded into the solid: concave closed-rim band: plane-contact radius 15.7162 exceeds the cap face half-extent 15 — the cove spills off the plate onto the adjacent walls (a multi-face concave interaction is a follow-on slice)",
	{"simple", "S8"}:           "curved (torus-arm) Plane∧Cylinder edge 20202 could not be welded into the solid: concave closed-rim band: plane-contact radius 31.1803 exceeds the cap face half-extent 30 — the cove spills off the plate onto the adjacent walls (a multi-face concave interaction is a follow-on slice)",
	{"simple", "T2"}:           "curved (torus-arm) Plane∧Cylinder edge 21508 could not be welded into the solid: concave closed-rim band: plane-contact radius 30.5279 exceeds the cap face half-extent 30 — the cove spills off the plate onto the adjacent walls (a multi-face concave interaction is a follow-on slice)",
	{"simple", "T5"}:           "cannot round an edge bordering a curved (geom.EllipticalCylinder) face — rounding a filleted or otherwise curved edge is not yet supported",
	{"simple", "T8"}:           "cannot round an edge bordering a curved (geom.BSplineSurface) face — rounding a filleted or otherwise curved edge is not yet supported",
	{"simple", "T9"}:           "builds but fails solid validity (empty decline reason — diagnosis not yet started)",
	{"simple", "U2"}:           "cannot round an edge bordering a curved (geom.EllipticalCylinder) face — rounding a filleted or otherwise curved edge is not yet supported",
	{"simple", "U5"}:           "cannot round an edge bordering a curved (cylinder) face — rounding a filleted or otherwise curved edge is not yet supported",
	{"simple", "U6"}:           "area +2.041% outside OCCT tolerance — geometry still wrong; must NOT be counted green",
	{"simple", "W4"}:           "curved (torus-arm) Plane∧Cylinder edge 27996 could not be welded into the solid: curved miter: assembled weld carries a boundary edge 0.1701 off its own face's surface (85.06% of the fillet radius, budget 10%; face 28053 (geom.Torus) bounded by edge 28050 (geom.LineSegment)) — a seam-sampling defect, not the near-boundary residual this weld already reconciles",
	{"simple", "W5"}:           "curved (torus-arm) Plane∧Cylinder edge 28153 could not be welded into the solid: arm rail bundle declined (geom.Torus): far vertex 28147: a second filleted edge 28148 also ends here (fillet-fillet interference / setback regime, out of scope)",
	{"simple", "W9"}:           "area +15.120% outside OCCT tolerance — geometry still wrong; must NOT be counted green",
	{"simple", "X2"}:           "radius 20.5 exceeds geometric maximum 19.999999999999996 on edge 28520 (available in-face width 20 on face 28523, dihedral 90.0°); reduce the radius or use a smaller value",
	{"simple", "X5"}:           "full-round 6-arm corner requires all-convex arms (edge 29785 is not convex)",
	{"simple", "X7"}:           "cannot round edge 29861 — it runs into an existing rounded (cylinder) face at its end; fillet these edges BEFORE the adjacent rounds, or select them together",
	{"simple", "X8"}:           "corner where 4 filleted edges meet a 4-face vertex has unequal dihedral angles between adjacent arms — the full-round K-gon patch is only certified for a dihedrally-symmetric corner (radius 10) — the free-form corner filling is not yet supported",
	{"tolblend_simple", "A2"}:  "mixed-radius corner where 4 filleted edges meet a 4-face vertex is not a supported blend (need 2 edges sharing a face, or 3 edges at a trihedral vertex)",
	{"tolblend_simple", "A9"}:  "mixed-radius corner where 5 filleted edges meet a 5-face vertex is not a supported blend (need 2 edges sharing a face, or 3 edges at a trihedral vertex)",
	{"tolblend_simple", "B1"}:  "corner where 6 filleted edges meet a 6-face vertex has no common tangent sphere (radius 2.5, plane residual 0.26317759250014205 > weld 8.325916698020573e-08) — the free-form corner filling is not yet supported",
	{"tolblend_simple", "B2"}:  "mixed-radius corner where 6 filleted edges meet a 6-face vertex is not a supported blend (need 2 edges sharing a face, or 3 edges at a trihedral vertex)",
	{"tolblend_simple", "B3"}:  "full-round 6-arm corner requires all-convex arms (edge 31496 is not convex)",
	{"tolblend_simple", "B7"}:  "cannot round an edge bordering a curved (geom.EllipticalCylinder) face — rounding a filleted or otherwise curved edge is not yet supported",
	{"tolblend_simple", "C2"}:  "cannot round an edge bordering a curved (geom.EllipticalCylinder) face — rounding a filleted or otherwise curved edge is not yet supported",
	{"tolblend_simple", "C4"}:  "cannot round edge 31801 at a 4-valent runout vertex {25 0 0} — no single crossing on far edge (edge 31787); reduce the radius or fillet the neighbours first",
	{"tolblend_simple", "C6"}:  "cannot round an edge bordering a curved (geom.BSplineSurface) face — rounding a filleted or otherwise curved edge is not yet supported",
	{"tolblend_simple", "C7"}:  "curved (cylinder-arm) Plane∧Cylinder edge 31873 could not be welded into the solid: trihedral corner needs 3 arms (got 1 at vertex 31871)",
	{"tolblend_simple", "C8"}:  "cannot round an edge bordering a curved (geom.BSplineSurface) face — rounding a filleted or otherwise curved edge is not yet supported",
	{"tolblend_simple", "D3"}:  "partial corner where 3 of 4 edges at the vertex are filleted needs its patch boundary closed through 2 gap face(s) bordering the surviving sharp edges at the vertex — that boundary-assembly capability is not yet built (the free-form/N-sided corner filling epic)",
	{"tolblend_simple", "D5"}:  "result is not a valid solid []",
	{"tolblend_simple", "E6"}:  "partial corner where 4 of 5 edges at the vertex are filleted needs its patch boundary closed through 2 gap face(s) bordering the surviving sharp edges at the vertex — that boundary-assembly capability is not yet built (the free-form/N-sided corner filling epic)",
	{"tolblend_simple", "E7"}:  "partial corner where 3 of 5 edges at the vertex are filleted needs its patch boundary closed through 4 gap face(s) bordering the surviving sharp edges at the vertex — that boundary-assembly capability is not yet built (the free-form/N-sided corner filling epic)",
	{"tolblend_simple", "E8"}:  "partial corner where 3 of 5 edges at the vertex are filleted needs its patch boundary closed through 2 gap face(s) bordering the surviving sharp edges at the vertex — that boundary-assembly capability is not yet built (the free-form/N-sided corner filling epic)",
}

// pendingCapabilityCount pins the size of the pending list so it can only SHRINK by review.
// 99 FAIL(faulty) + 5 FAIL(area), measured at 09a9b2d1 on 2026-08-01.
const pendingCapabilityCount = 104

// pendingCapabilityReason returns the not-yet-built reason for a case and whether it is pending.
//
// Example:
//
//	if why, pending := pendingCapabilityReason(r); pending { t.Skipf("pending: %s", why) }
func pendingCapabilityReason(r Record) (string, bool) {
	reason, pending := pendingCapability[quarantineKey{grid: r.Grid, name: r.Case}]
	return reason, pending
}
