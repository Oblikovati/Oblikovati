// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// setbackFilsForCase solves a wired corpus case's pick edge and applies the rail-termination setback,
// returning the corrected fils exactly as filletResolvedEdges does before it rebuilds the body.
func setbackFilsForCase(t *testing.T, rel string) (*topo.Body, []edgeFillet) {
	t.Helper()
	b := importCorpusSolid(t, rel)
	fils := solvedFilsForCase(t, b, rel)
	if err := applyRunoutSetback(fils); err != nil {
		t.Fatalf("%s applyRunoutSetback: %v", rel, err)
	}
	return b, fils
}

// TestRunoutSetbackPiercesFarPlaneV5 is the pierce gate: after the setback, V5's fan-end flank rails
// (fan.ta/fan.tb, copied from the corrected corner) must land on OCCT's runout vertices RV_3 and
// RV_12 — the line∩far-plane pierce — not the apex-vertex axial projection the raw ta/tb overshoots
// to. Fails RED against today's ta/tb (station 110.38, ~3.2 / ~5.4 beyond the pierce).
func TestRunoutSetbackPiercesFarPlaneV5(t *testing.T) {
	t.Parallel()
	_, fils := setbackFilsForCase(t, "simple/V5")
	fans, _ := classifyEndCorners(fils)
	if len(fans) != 1 {
		t.Fatalf("V5: got %d fans, want 1", len(fans))
	}
	f := fans[0]
	rv3 := math.P3(39.5842, 88.1164, 51.7052)
	rv12 := math.P3(40.0319, 85.8487, 47.3618)
	if d := float64(f.ta.DistanceTo(rv3)); d > 1e-3 {
		t.Errorf("V5 fan.ta = %v, want RV_3 %v (dist %.4g)", f.ta, rv3, d)
	}
	if d := float64(f.tb.DistanceTo(rv12)); d > 1e-3 {
		t.Errorf("V5 fan.tb = %v, want RV_12 %v (dist %.4g)", f.tb, rv12, d)
	}
}

// TestRunoutSetbackPiercesFarPlaneV1 gates the symmetric case: V1's fan end (apex) and its trihedral
// start end both set back — the fan rails to (47.9657,67.1519,47.9657)/(52.0343,67.1519,52.0343), the
// trihedral start rails to (0,0,95.9313)/(4.0687,0,100) — all four OCCT runout vertices.
func TestRunoutSetbackPiercesFarPlaneV1(t *testing.T) {
	t.Parallel()
	_, fils := setbackFilsForCase(t, "simple/V1")
	ef := fils[0]
	fanEnd, triEnd := ef.c1, ef.c0 // probe confirmed: c1 = valence-4 fan, c0 = valence-3 trihedral
	wantFanA := math.P3(47.9657, 67.1519, 47.9657)
	wantFanB := math.P3(52.0343, 67.1519, 52.0343)
	wantTriA := math.P3(0, 0, 95.9313)
	wantTriB := math.P3(4.0687, 0, 100)
	assertNear(t, "V1 fan.ta", fanEnd.ta, wantFanA)
	assertNear(t, "V1 fan.tb", fanEnd.tb, wantFanB)
	assertNear(t, "V1 tri.ta", triEnd.ta, wantTriA)
	assertNear(t, "V1 tri.tb", triEnd.tb, wantTriB)
}

func assertNear(t *testing.T, name string, got, want math.Point3) {
	t.Helper()
	if d := float64(got.DistanceTo(want)); d > 1e-3 {
		t.Errorf("%s = %v, want %v (dist %.4g)", name, got, want, d)
	}
}

// TestRunoutSetbackRailWeldsCapV5 is the single-source coupling gate: after the setback, the cylinder
// A-rail's end (the fan corner's ta) and B-rail's start (its tb) must equal the fan cap's FIRST and
// LAST piece endpoints. If the rail read a raw ta/tb while the cap read the pierced fan.ta/tb, they
// would disagree and the cylinder loop would open (a non-solid). This proves both consumers derive
// from the one corrected corner.
func TestRunoutSetbackRailWeldsCapV5(t *testing.T) {
	t.Parallel()
	b, fils := setbackFilsForCase(t, "simple/V5")
	fans, _ := classifyEndCorners(fils)
	_, caps := buildSpreadMaps(fans, b)
	ef := fils[0]
	checked := 0
	for _, c := range []corner{ef.c0, ef.c1} {
		if _, ok := fanForEndCorner(ef, c); !ok {
			continue
		}
		segs := capEndSegs(caps[c.vertex.ID()])
		// cylinderFace lays down the A-rail (…→c.ta) then the cap (c.ta→…→c.tb) then the B-rail
		// (c.tb→…): the rail ends must be exactly the cap's first from / last to.
		if d := float64(c.ta.DistanceTo(segs[0].from)); d > 1e-9 {
			t.Errorf("A-rail end c.ta %v != first cap piece start %v (dist %.3g) — loop opens", c.ta, segs[0].from, d)
		}
		if d := float64(c.tb.DistanceTo(segs[len(segs)-1].to)); d > 1e-9 {
			t.Errorf("B-rail start c.tb %v != last cap piece end %v (dist %.3g) — loop opens", c.tb, segs[len(segs)-1].to, d)
		}
		checked++
	}
	if checked != 1 {
		t.Fatalf("V5 checked %d fan corners, want 1", checked)
	}
}

// TestRunoutSetbackKeepsInteriorV5 is the interior-unchanged invariant: the setback moves ONLY the two
// flank rail ends. Every far-edge split (RV_9/10/11), a function of center/axis/radius/farEdges alone,
// must be byte-identical before and after the setback — and still equal to OCCT's runout vertices.
func TestRunoutSetbackKeepsInteriorV5(t *testing.T) {
	t.Parallel()
	b := importCorpusSolid(t, "simple/V5")
	fils := solvedFilsForCase(t, b, "simple/V5")
	rawFans, _ := classifyEndCorners(fils)
	rawSp, err := solveRunoutSpread(rawFans[0])
	if err != nil {
		t.Fatalf("raw solve: %v", err)
	}
	if err := applyRunoutSetback(fils); err != nil {
		t.Fatalf("setback: %v", err)
	}
	setFans, _ := classifyEndCorners(fils)
	setSp, err := solveRunoutSpread(setFans[0])
	if err != nil {
		t.Fatalf("setback solve: %v", err)
	}
	if len(rawSp.splits) != len(setSp.splits) {
		t.Fatalf("split count changed %d -> %d", len(rawSp.splits), len(setSp.splits))
	}
	for id, p := range rawSp.splits {
		if d := float64(p.DistanceTo(setSp.splits[id])); d > 1e-9 {
			t.Errorf("far-edge %d split moved by %.3g — the setback disturbed the runout interior", id, d)
		}
	}
	for _, rv := range []math.Point3{
		math.P3(41.8689, 89.7882, 50.4648), // RV_9
		math.P3(42.2209, 89.8985, 50.1647), // RV_10
		math.P3(42.1898, 89.3411, 49.4004), // RV_11
	} {
		if !splitNear(setSp, rv) {
			t.Errorf("no far-edge split near OCCT runout vertex %v after setback", rv)
		}
	}
}

// splitNear reports whether any far-edge split of sp sits within 1e-3 of p.
func splitNear(sp runoutSpread, p math.Point3) bool {
	for _, s := range sp.splits {
		if s.DistanceTo(p) < 1e-3 {
			return true
		}
	}
	return false
}

// TestRunoutSetbackSolidV5 and V1 are the end-to-end weld/solid invariant: driving production
// FilletEdges through the setback must still yield a valid solid whose every edge is used exactly
// twice and from which the >3-valent apex vertex has been dropped (the runout replaces it) — proving
// the moved rail ends did not open the shell.
func TestRunoutSetbackSolidV5(t *testing.T) {
	t.Parallel()
	assertRunoutSolid(t, "simple/V5", math.P3(42.2618, 90.6308, 50.0))
}

func TestRunoutSetbackSolidV1(t *testing.T) {
	t.Parallel()
	assertRunoutSolid(t, "simple/V1", math.P3(50, 70, 50))
}

func assertRunoutSolid(t *testing.T, rel string, apex math.Point3) {
	t.Helper()
	b := importCorpusSolid(t, rel)
	pa, pb, r, ok := filletPickForCase(rel)
	if !ok {
		t.Fatalf("%s not wired", rel)
	}
	e := edgeBetween(t, b, vertexNear(t, b, pa), vertexNear(t, b, pb))
	res, err := FilletEdges(b, [][]byte{e.ReferenceKey()}, r)
	if err != nil {
		t.Fatalf("%s FilletEdges: %v", rel, err)
	}
	if !res.IsSolid() {
		t.Fatalf("%s result not marked solid", rel)
	}
	if rep := Validate(res); !rep.Valid {
		t.Fatalf("%s result invalid: %v", rel, rep.Issues)
	}
	open := 0
	for _, ed := range res.Edges() {
		if len(ed.Uses()) != 2 {
			open++
		}
	}
	if open != 0 {
		t.Errorf("%s left %d open edges after setback", rel, open)
	}
	for _, v := range res.Vertices() {
		if v.Point().DistanceTo(apex) < 1e-6 {
			t.Errorf("%s apex vertex %v survived — the runout must drop it", rel, apex)
		}
	}
}
