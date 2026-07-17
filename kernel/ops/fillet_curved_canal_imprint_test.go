// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// W3c/F2 — the canal FAR-END host imprints, oracle-pinned on the REAL N7 STEP body. F1 gave the three arms
// their geometric far termini (building on the real body); F2 fixes the HOST side: the y=30 verbatim
// splice, the z=80 co-circular extension arc + arc bite, the x=80 collinear extension segment + spiric
// bite, and the wall's extension-augmented far path. These tests drive the REAL imported body + real edges
// with the fixture arm SURFACES (the oracle-correct s_10 spine at x=55; the production arm-surface build
// mirrors it — see farrunout-f2-report.md), so F_far and every loop are real topology fed correct termini.

// n7RealHost holds the real-body host-retrim inputs: the imported body, the corner weld, the real-bound
// arms, their per-arm bundles (real termini + far-end extensions), the tagged boundaries and the Rolls.
type n7RealHost struct {
	body       *topo.Body
	w          cornerWeld
	arms       []edgeFillet
	bundles    []canalArmBundle
	boundaries canalBoundaries
	rolls      []geom.Surface
	res        Resolution
}

// n7RealHostInputs imports the real N7 body ONCE and binds the fixture arms to its edges/faces (keeping the
// oracle-correct fixture surfaces), then builds the bundles at the real reflected centres — all consistent
// with the SAME body, so canalHostFaces' pointer identity between a host face and bundle.hosts holds.
func n7RealHostInputs(t *testing.T) n7RealHost {
	t.Helper()
	w, fix, res := n7CornerFill(t)
	_, boundaries, _, _ := n7CanalWeldInputs(t, w, fix, res)
	_, _, rolls := n7CanalBiteInputs(t, w, fix, res)
	body := importCorpusSolid(t, "simple/N7")
	corner := vertexNear(t, body, math.P3(50, 0, 10))
	arms := make([]edgeFillet, len(fix))
	for i := range fix {
		arms[i] = bindArmToRealEdge(t, fix[i], corner)
	}
	scale := tangentCornerScale(w, arms)
	centres, ok, _ := reflectedArmCentres(w, arms, scale, res)
	if !ok {
		t.Fatal("real-body reflectedArmCentres unresolved")
	}
	bundles, ok := canalArmBundles(arms, w, centres, scale, res)
	if !ok {
		t.Fatal("real-body canalArmBundles declined")
	}
	return n7RealHost{body, w, arms, bundles, boundaries, rolls, res}
}

// armBundle returns the bundle for arms[i] (bundles are index-parallel to arms).
func (h n7RealHost) armBundle(i int) canalArmBundle { return h.bundles[i] }

// farFaceOf finds the real topo.Face that is arm i's terminating face F_far (the plane at its far vertex
// matching canalFarFace), so the imprint splice can be driven on the real loop.
func (h n7RealHost) farFaceOf(t *testing.T, i int) *topo.Face {
	t.Helper()
	wi := cornerWeld{center: h.w.center, radius: h.w.radius}
	ffar, ok := canalFarFace(h.arms[i], wi)
	if !ok {
		t.Fatalf("arm %d: canalFarFace unresolved", i)
	}
	far := farEndVertex(h.arms[i].edge, wi.center)
	for _, f := range facesAround(far) {
		if pl, isPl := f.Geometry().(geom.Plane); isPl && samePlaneGeom(pl, ffar, h.res.Weld()*h.w.radius) {
			return f
		}
	}
	t.Fatalf("arm %d: no real F_far face matches the plane", i)
	return nil
}

// wallFace returns the R=50 cylinder host face (shared by the two wall-sharing arms).
func (h n7RealHost) wallFace(t *testing.T) *topo.Face {
	t.Helper()
	for _, a := range h.arms {
		for _, f := range [2]*topo.Face{a.a, a.b} {
			if _, ok := f.Geometry().(geom.Cylinder); ok {
				return f
			}
		}
	}
	t.Fatal("no cylinder wall host among the arms")
	return nil
}

// samePlaneGeom reports whether two planes coincide: parallel normals and one origin on the other plane.
func samePlaneGeom(a, b geom.Plane, tol float64) bool {
	na, nb := a.Normal(), b.Normal()
	if stdmath.Abs(float64(na.Dot(nb))) < 1-sinFloor {
		return false
	}
	return stdmath.Abs(float64(a.Origin.VectorTo(b.Origin).Dot(na))) <= tol
}

// TestCanalImprint_Y30Verbatim pins result_10 (s_10 F_far = y=30): BOTH terminal feet land on the loop, so
// the arm carries NO extension (extHost nil) and the imprint splices the terminal VERBATIM — byte-identical
// to farRunoutFace's single-bite splice (the derivation's "verbatim B3 spliceCornerBite").
func TestCanalImprint_Y30Verbatim(t *testing.T) {
	h := n7RealHostInputs(t)
	i := cylinderArmAlong(t, h.arms, math.V3(0, 1, 0)) // s_10: axis ŷ
	b := h.armBundle(i)
	if b.extHost != nil {
		t.Fatalf("s_10 must carry NO extension (both feet on the y=30 loop); got extHost=%v", b.extHost)
	}
	f := h.farFaceOf(t, i)
	tol := h.res.Weld() * h.w.radius
	ff, reason := canalImprintFace(f, h.bundles, tol)
	if reason != "" {
		t.Fatalf("y=30 imprint declined: %s", reason)
	}
	assertClosedImprint(t, ff, "y=30")
	// Verbatim: the chain splice with the single terminal equals farRunoutFace's single-bite splice.
	want, ok := farRunoutFace(f, []endSeg{b.far}, tol)
	if !ok {
		t.Fatal("farRunoutFace must splice the y=30 terminal (both feet on-loop)")
	}
	assertSameLoopPoints(t, ff, want, "y=30 verbatim")
}

// TestCanalImprint_Z80ArcExtension pins result_4 (s_4 F_far = z=80 ⊥ spine): the wall foot runs off the
// loop, so a CO-CIRCULAR extension arc (on the wall∩{z=80} circle R=50) sweeps asin(1/9) from the far vertex
// (50,0,80) to the wall foot; the imprint splices [ext, arc terminal] and closes.
func TestCanalImprint_Z80ArcExtension(t *testing.T) {
	h := n7RealHostInputs(t)
	i := cylinderArmAlong(t, h.arms, math.V3(0, 0, 1)) // s_4: axis ẑ
	b := h.armBundle(i)
	tol := h.res.Weld() * h.w.radius
	if b.extHost == nil {
		t.Fatal("s_4 must carry a z=80 extension (wall foot off the loop)")
	}
	assertPointNear(t, "z=80 ext far vertex q", b.ext.from, math.P3(50, 0, 80), tol)
	assertPointNear(t, "z=80 ext wall foot", b.ext.to, math.P3(50-50.0/9, 50-(10.0/9)*stdmath.Sqrt(2000), 80), tol)
	arc, ok := b.ext.curve.(geom.Arc3d)
	if !ok || !b.ext.arc {
		t.Fatalf("z=80 extension must be a co-circular Arc3d; got %T (arc=%v)", b.ext.curve, b.ext.arc)
	}
	assertScalarNear(t, "z=80 ext sweep asin(1/9)", stdmath.Abs(arc.SweepAngle), stdmath.Asin(1.0/9), tol)
	assertPointNear(t, "z=80 ext arc centre", arc.Center, math.P3(50, 50, 80), tol)
	ff, reason := canalImprintFace(h.farFaceOf(t, i), h.bundles, tol)
	if reason != "" {
		t.Fatalf("z=80 imprint declined: %s", reason)
	}
	assertClosedImprint(t, ff, "z=80")
}

// TestCanalImprint_X80SpiricExtension pins result_2 (s_5 F_far = x=80 ∥ axis): a COLLINEAR extension segment
// of length r on the wall∩{x=80} ruling from the far vertex (80,10,10) to the wall foot (80,10,5), and the
// bite is the SPIRIC section (a geom.SpiricArc, threaded UNREVERSED — never a chord). The imprint closes.
func TestCanalImprint_X80SpiricExtension(t *testing.T) {
	h := n7RealHostInputs(t)
	i := torusArmIndex(t, h.arms) // s_5 (torus)
	b := h.armBundle(i)
	tol := h.res.Weld() * h.w.radius
	if b.extHost == nil {
		t.Fatal("s_5 must carry an x=80 extension (wall foot off the loop)")
	}
	assertPointNear(t, "x=80 ext far vertex q", b.ext.from, math.P3(80, 10, 10), tol)
	assertPointNear(t, "x=80 ext wall foot", b.ext.to, math.P3(80, 10, 5), tol)
	if b.ext.arc || b.ext.curve != nil {
		t.Fatalf("x=80 extension must be a collinear straight segment; got curve %T (arc=%v)", b.ext.curve, b.ext.arc)
	}
	assertScalarNear(t, "x=80 ext length r", float64(b.ext.from.DistanceTo(b.ext.to)), 5, tol)
	if _, ok := b.far.curve.(geom.SpiricArc); !ok {
		t.Fatalf("s_5 terminal must be a geom.SpiricArc (not chorded/reversed); got %T", b.far.curve)
	}
	ff, reason := canalImprintFace(h.farFaceOf(t, i), h.bundles, tol)
	if reason != "" {
		t.Fatalf("x=80 imprint declined: %s", reason)
	}
	assertClosedImprint(t, ff, "x=80")
	assertSpiricInLoop(t, ff, "x=80")
}

// TestCanalCloseFar_WallAugmentedLoop is the F1-floor-gone deliverable: the wall host bite CLOSES on the
// window loop AUGMENTED by the two extension edges (the F1 floor "far span will not close — outer ends off
// the bitten loop A=5.0 B=5.5642" is gone). The wall retrims to a valid face.
func TestCanalCloseFar_WallAugmentedLoop(t *testing.T) {
	h := n7RealHostInputs(t)
	wall := h.wallFace(t)
	ff, reason := canalHostBite(wall, h.bundles, h.boundaries, h.rolls, h.w, h.res)
	if reason != "" {
		t.Fatalf("wall host bite must CLOSE on the augmented loop (F1 floor must be gone); declined: %s", reason)
	}
	assertClosedImprint(t, ff, "wall")
	if _, ok := ff.surface.(geom.Cylinder); !ok {
		t.Fatalf("wall retrim surface is %T, want geom.Cylinder", ff.surface)
	}
}

// TestCanalExtension_SharedEdgeIdentity is the shared-edge gate: the SAME extension endSeg object closes the
// F_far imprint face AND the wall's far path — its two endpoints (far vertex q, wall foot) are point-
// identical on both, and both extensions land on the wall (extHost == wall).
func TestCanalExtension_SharedEdgeIdentity(t *testing.T) {
	h := n7RealHostInputs(t)
	wall := h.wallFace(t)
	exts := extensionsOnHost(wall, h.bundles)
	if len(exts) != 2 {
		t.Fatalf("the wall must carry BOTH far-end extensions (s_4 z=80, s_5 x=80); got %d", len(exts))
	}
	for _, arm := range []int{cylinderArmAlong(t, h.arms, math.V3(0, 0, 1)), torusArmIndex(t, h.arms)} {
		b := h.armBundle(arm)
		if b.extHost != wall {
			t.Fatalf("arm %d extension host is %v, want the wall face %v", arm, b.extHost, wall)
		}
		// The wall's copy of this extension is the SAME object (endpoints residual exactly 0).
		matched := false
		for _, e := range exts {
			if e.from == b.ext.from && e.to == b.ext.to {
				matched = true
			}
		}
		if !matched {
			t.Fatalf("arm %d extension %v→%v is not shared point-identically with the wall's far path", arm, b.ext.from, b.ext.to)
		}
	}
}

// TestCanalImprint_NoExtensionMutationDeclines is the discriminator (do-no-harm): DROP the extension edges
// and the z=80/x=80 imprint splices DECLINE (a bite foot off the loop) and the wall far span DECLINES (the
// F1-era floor) — proving the extensions are load-bearing, not cosmetic.
func TestCanalImprint_NoExtensionMutationDeclines(t *testing.T) {
	h := n7RealHostInputs(t)
	tol := h.res.Weld() * h.w.radius
	bare := stripExtensions(h.bundles)
	for _, name := range []struct {
		label string
		idx   int
	}{{"z=80", cylinderArmAlong(t, h.arms, math.V3(0, 0, 1))}, {"x=80", torusArmIndex(t, h.arms)}} {
		f := h.farFaceOf(t, name.idx)
		if _, reason := canalImprintFace(f, bare, tol); reason == "" {
			t.Fatalf("%s imprint must DECLINE without its extension (wall foot off the loop)", name.label)
		}
	}
	wall := h.wallFace(t)
	if _, reason := canalHostBite(wall, bare, h.boundaries, h.rolls, h.w, h.res); reason == "" {
		t.Fatal("the wall far span must DECLINE without the extensions (the F1-era floor)")
	}
}

// TestChainBiteArea_SpiricSampledNotChorded proves the spiric bite's area contribution is SAMPLED, not
// chorded: the chain area with the true SpiricArc bulge differs measurably from the chord (straight-segment)
// approximation and matches a fine-sampled reference — so the smaller-area splice pick stays principled.
func TestChainBiteArea_SpiricSampledNotChorded(t *testing.T) {
	h := n7RealHostInputs(t)
	i := torusArmIndex(t, h.arms)
	b := h.armBundle(i)
	sa, ok := b.far.curve.(geom.SpiricArc)
	if !ok {
		t.Fatalf("s_5 terminal is %T, want geom.SpiricArc", b.far.curve)
	}
	span := []endSeg{{from: b.far.to, to: b.far.from}} // the chord span closing the spiric bite
	sampled := chainBiteArea(span, []endSeg{b.far})
	chord := chainBiteArea(span, []endSeg{{from: b.far.from, to: b.far.to}}) // strip the curve → chord
	if stdmath.Abs(sampled-chord) < 1e-3 {
		t.Fatalf("spiric area (%.6f) must differ from its chord (%.6f) — sampling is not engaged", sampled, chord)
	}
	ref := fineSpiricRingArea(sa, span[0].from, span[0].to)
	if d := stdmath.Abs(sampled - ref); d > 0.05 {
		t.Fatalf("sampled spiric area %.6f is %.3e off the fine reference %.6f (biteArcSamples too coarse?)", sampled, d, ref)
	}
	t.Logf("spiric bite area: sampled=%.6f chord=%.6f fine-ref=%.6f", sampled, chord, ref)
}

// --- helpers ---------------------------------------------------------------

// assertClosedImprint asserts a retrimmed face has a first loop that is a closed ≥4-point ring.
func assertClosedImprint(t *testing.T, ff filletFace, name string) {
	t.Helper()
	if len(ff.loops) == 0 || len(ff.loops[0].pts) < 4 {
		t.Fatalf("%s retrim must yield a closed ≥4-point bite loop; got %d loops", name, len(ff.loops))
	}
}

// assertSpiricInLoop asserts the retrimmed loop carries the UNREVERSED SpiricArc object (never a chord nor
// a reversedCurve3 wrapper that would erase its concrete type).
func assertSpiricInLoop(t *testing.T, ff filletFace, name string) {
	t.Helper()
	for _, c := range ff.loops[0].curves {
		if _, ok := c.(geom.SpiricArc); ok {
			return
		}
	}
	t.Fatalf("%s loop must carry the unreversed geom.SpiricArc bite; none found among %d curves", name, len(ff.loops[0].curves))
}

// stripExtensions returns a copy of the bundles with every far-end extension removed (the mutation).
func stripExtensions(bundles []canalArmBundle) []canalArmBundle {
	out := make([]canalArmBundle, len(bundles))
	copy(out, bundles)
	for i := range out {
		out[i].ext, out[i].extHost = endSeg{}, nil
	}
	return out
}

// fineSpiricRingArea is the Newell area of the triangle-like region the spiric encloses with its chord,
// sampled at high resolution — the independent reference the coarse biteArcSamples area is checked against.
func fineSpiricRingArea(sa geom.SpiricArc, from, to math.Point3) float64 {
	ring := []math.Point3{from}
	fwd := float64(sa.PointAt(0).DistanceTo(from)) <= float64(sa.PointAt(1).DistanceTo(from))
	const n = 2000
	for i := 1; i < n; i++ {
		t := float64(i) / float64(n)
		if !fwd {
			t = 1 - t
		}
		ring = append(ring, sa.PointAt(t))
	}
	ring = append(ring, to)
	return float64(newellNormal(ring).Length()) / 2
}
