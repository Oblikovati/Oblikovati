// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/analysis"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
	"oblikovati.org/persistence"
)

// TestParameterTranslationRoundTrips translates a part with a user parameter and checks
// the reopened .opd carries it as a native Oblikovati parameter (L = 10 mm = 1 cm).
func TestParameterTranslationRoundTrips(t *testing.T) {
	out := filepath.Join(t.TempDir(), "paramL.opd")
	if _, err := FromInventor(readCorpus(t, "param_L10.ipt"), out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	p, ok := def.Parameters().ByName("L")
	if !ok {
		t.Fatalf("reopened part has no parameter %q", "L")
	}
	if math.Abs(p.NominalValue()-1.0) > 1e-9 { // database units: cm
		t.Errorf("parameter L = %.4f cm, want 1.0", p.NominalValue())
	}
}

// TestSketchTranslationRoundTrips translates the line and circle sketches and checks the
// reopened .opd carries a native sketch (entity fidelity is covered by the decode tests
// and the recipe YAML; this asserts the sketch persists and reloads).
func TestSketchTranslationRoundTrips(t *testing.T) {
	for _, file := range []string{"sketch_line.ipt", "sketch_circle.ipt"} {
		out := filepath.Join(t.TempDir(), "s.opd")
		if _, err := FromInventor(readCorpus(t, file), out); err != nil {
			t.Fatalf("%s: FromInventor: %v", file, err)
		}
		ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
		reopened, err := ws.Open(out, true)
		if err != nil {
			t.Fatalf("%s: reopen: %v", file, err)
		}
		def := reopened.Content().(*compdef.PartComponentDefinition)
		if n := def.Sketches().Count(); n != 1 {
			t.Errorf("%s: reopened part has %d sketches, want 1", file, n)
		}
	}
}

// TestExtrudeTranslationRebuildsSolid translates the cylinder (circle sketch + extrude)
// PARAMETRICALLY — the recipe carries the sketch and extrude feature, and the kernel
// rebuilds the solid on open. Volume ~= pi*r^2*h = pi*1^2*2 (a small deficit is the
// curved-surface tessellation bias).
func TestExtrudeTranslationRebuildsSolid(t *testing.T) {
	out := filepath.Join(t.TempDir(), "cyl.opd")
	if _, err := FromInventor(readCorpus(t, "15_cylinder.ipt"), out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	if def.Features().Count() == 0 {
		t.Fatalf("reopened cylinder has no features (not rebuilt parametrically)")
	}
	mp := analysis.MassPropertiesOf(def.SurfaceBodies().All(), 1, types.MassPropertiesHigh)
	want := 2 * math.Pi * 1000 // pi*r^2*h in mm^3 (r=10mm, h=20mm)
	if math.Abs(mp.VolumeMm3-want) > 0.02*want {
		t.Errorf("cylinder volume = %.1f mm^3, want ~%.1f (within 2%%)", mp.VolumeMm3, want)
	}
}

// TestRevolveTranslationRebuildsSolid translates the revolve (rectangle profile revolved
// 360° about X) parametrically and checks the tube volume: pi*(R^2-r^2)*L =
// pi*(1.5^2-0.5^2)*2 = 4*pi cm^3 (small deficit = curved-surface tessellation).
func TestRevolveTranslationRebuildsSolid(t *testing.T) {
	out := filepath.Join(t.TempDir(), "rev.opd")
	if _, err := FromInventor(readCorpus(t, "16_revolve.ipt"), out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	mp := analysis.MassPropertiesOf(def.SurfaceBodies().All(), 1, types.MassPropertiesHigh)
	want := 4 * math.Pi * 1000 // 4*pi cm^3 in mm^3
	if math.Abs(mp.VolumeMm3-want) > 0.02*want {
		t.Errorf("revolve tube volume = %.1f mm^3, want ~%.1f (within 2%%)", mp.VolumeMm3, want)
	}
}

// TestBooleanTranslationRebuildsSolid translates the multi-feature boolean parts and
// checks the reopened volume: cut removes the inner column (8 - 2*1*0.5 = 7 cm^3); join
// adds the non-overlapping part of the second box (8 + 2*1*1 = 10 cm^3).
func TestBooleanTranslationRebuildsSolid(t *testing.T) {
	cases := []struct {
		file    string
		wantMm3 float64
	}{
		{"17_box_cut.ipt", 7000},
		{"14_box_two.ipt", 10000},
	}
	for _, tc := range cases {
		out := filepath.Join(t.TempDir(), "b.opd")
		if _, err := FromInventor(readCorpus(t, tc.file), out); err != nil {
			t.Fatalf("%s: FromInventor: %v", tc.file, err)
		}
		ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
		reopened, err := ws.Open(out, true)
		if err != nil {
			t.Fatalf("%s: reopen: %v", tc.file, err)
		}
		def := reopened.Content().(*compdef.PartComponentDefinition)
		mp := analysis.MassPropertiesOf(def.SurfaceBodies().All(), 1, types.MassPropertiesHigh)
		if math.Abs(mp.VolumeMm3-tc.wantMm3) > 1 {
			t.Errorf("%s: volume = %.1f mm^3, want %.1f", tc.file, mp.VolumeMm3, tc.wantMm3)
		}
	}
}

// TestNonConvexProfileRebuildsSolid translates the L-profile (a non-convex 6-corner
// sketch, extruded 1 cm) parametrically and checks the reopened volume: the L area is
// 4*1 + 1*2 = 6 cm^2, so the solid is 6 cm^3. A convex-hull sketch would over-fill the
// notch and measure more, so an exact 6000 mm^3 proves the endpoint-graph loop decode.
func TestNonConvexProfileRebuildsSolid(t *testing.T) {
	out := filepath.Join(t.TempDir(), "l.opd")
	if _, err := FromInventor(readCorpus(t, "18_lprofile.ipt"), out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	if def.Features().Count() == 0 {
		t.Fatalf("reopened L-profile has no features (not rebuilt parametrically)")
	}
	mp := analysis.MassPropertiesOf(def.SurfaceBodies().All(), 1, types.MassPropertiesHigh)
	if math.Abs(mp.VolumeMm3-6000) > 1 {
		t.Errorf("L-profile volume = %.1f mm^3, want 6000", mp.VolumeMm3)
	}
}

// TestHoleTranslationRebuildsSolid translates the drilled-hole parts parametrically (base
// box 4x2x2 = 16 cm^3 minus a Ø1 cm bore) and checks the reopened volume: a through bore
// removes pi*0.5^2*2 = 1.5708 cm^3 (→ 14.4292), a blind depth-1 bore pi*0.5^2*1 = 0.7854
// (→ 15.2146). The hole is a real HoleFeature (exact cylinder wall), so volumes are tight.
func TestHoleTranslationRebuildsSolid(t *testing.T) {
	cases := []struct {
		file    string
		wantMm3 float64
	}{
		{"19_box_hole.ipt", 16000 - math.Pi*0.25*2*1000},
		{"20_box_hole_blind.ipt", 16000 - math.Pi*0.25*1*1000},
	}
	for _, tc := range cases {
		out := filepath.Join(t.TempDir(), "h.opd")
		if _, err := FromInventor(readCorpus(t, tc.file), out); err != nil {
			t.Fatalf("%s: FromInventor: %v", tc.file, err)
		}
		ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
		reopened, err := ws.Open(out, true)
		if err != nil {
			t.Fatalf("%s: reopen: %v", tc.file, err)
		}
		def := reopened.Content().(*compdef.PartComponentDefinition)
		if def.Features().Count() < 2 {
			t.Fatalf("%s: reopened part has %d features, want >=2 (extrude + hole)", tc.file, def.Features().Count())
		}
		mp := analysis.MassPropertiesOf(def.SurfaceBodies().All(), 1, types.MassPropertiesHigh)
		if math.Abs(mp.VolumeMm3-tc.wantMm3) > 0.005*tc.wantMm3 {
			t.Errorf("%s: volume = %.1f mm^3, want ~%.1f", tc.file, mp.VolumeMm3, tc.wantMm3)
		}
	}
}

// TestRectPatternTranslationRebuildsSolid translates the pocket + rectangular-pattern part
// parametrically: a 6x4x1 = 24 cm^3 box with a 0.6x0.6 through-pocket patterned 3x along X
// removes 3*(0.6*0.6*1) = 1.08 cm^3 → 22.92. Proves the pattern replicates the cut.
func TestRectPatternTranslationRebuildsSolid(t *testing.T) {
	out := filepath.Join(t.TempDir(), "pat.opd")
	if _, err := FromInventor(readCorpus(t, "21_pocket_rect.ipt"), out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	if def.Features().Count() < 3 {
		t.Fatalf("reopened part has %d features, want >=3 (box + pocket + pattern)", def.Features().Count())
	}
	mp := analysis.MassPropertiesOf(def.SurfaceBodies().All(), 1, types.MassPropertiesHigh)
	if want := 24000.0 - 3*0.36*1000; math.Abs(mp.VolumeMm3-want) > 1 {
		t.Errorf("patterned volume = %.1f mm^3, want %.1f", mp.VolumeMm3, want)
	}
}

// TestCircPatternTranslationRebuildsSolid translates the disk + circular-pattern part: a
// r=3 x1 disk (π·9 = 28.274 cm^3) with a Ø1 through-pocket patterned 6× over 360° about Z
// removes 6·π·0.5²·1 = 4.712 cm^3 → 23.562. A small deficit is curved-surface tessellation.
func TestCircPatternTranslationRebuildsSolid(t *testing.T) {
	out := filepath.Join(t.TempDir(), "circ.opd")
	if _, err := FromInventor(readCorpus(t, "22_pocket_circ.ipt"), out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	if def.Features().Count() < 3 {
		t.Fatalf("reopened part has %d features, want >=3 (disk + pocket + pattern)", def.Features().Count())
	}
	mp := analysis.MassPropertiesOf(def.SurfaceBodies().All(), 1, types.MassPropertiesHigh)
	want := (math.Pi*9 - 6*math.Pi*0.25) * 1000 // mm^3
	if math.Abs(mp.VolumeMm3-want) > 0.02*want {
		t.Errorf("circular-patterned volume = %.1f mm^3, want ~%.1f (within 2%%)", mp.VolumeMm3, want)
	}
}

// TestCircPatternDoesNotReplicateBaseSolid guards the pattern-on-base fix: a Ø46 plate whose
// only feature is the base extrude, with a circular pattern authored to replicate its bolt-hole
// (already baked into the profile's inner loops), must NOT stamp the whole plate 5× — a centred
// base disk rotated about its own axis makes 5 coincident copies, inflating the volume to 5×2098.
// The pattern's source must be a cut/join, never the base, so with no such feature the pattern is
// skipped and one plate stands (~2098 mm³ vs Inventor's 2122). Corpus-gated: this exact geometry
// only exists in the real part (no generated fixture reproduces the centred-disk degeneracy), so
// the test skips where the corpus is absent (CI) and runs on a dev checkout of the ReelToReel set.
// Point IPT_CORPUS at the Mechanical directory to enable it.
func TestCircPatternDoesNotReplicateBaseSolid(t *testing.T) {
	dir := os.Getenv("IPT_CORPUS")
	if dir == "" {
		dir = `P:\ReelToReel\Mechanical`
	}
	data, err := os.ReadFile(filepath.Join(dir, "SmartKnobConnectingPlate.ipt"))
	if err != nil {
		t.Skipf("corpus part not available (%v); set IPT_CORPUS to the ReelToReel Mechanical dir", err)
	}
	out := filepath.Join(t.TempDir(), "plate.opd")
	if _, err := FromInventor(data, out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	bodies := def.SurfaceBodies().All()
	if len(bodies) != 1 {
		t.Fatalf("built %d bodies, want 1 (a circular pattern must not replicate the base solid)", len(bodies))
	}
	mp := analysis.MassPropertiesOf(bodies, 1, types.MassPropertiesLow)
	const oracle = 2122.0 // Inventor STL volume, mm^3
	if math.Abs(mp.VolumeMm3-oracle) > 0.05*oracle {
		t.Errorf("plate volume = %.0f mm^3, want ~%.0f (within 5%%)", mp.VolumeMm3, oracle)
	}
}

// TestAssemblyIsReportedNotMisTranslated confirms an .iam is detected and refused with a
// structured message (component + occurrence count) rather than silently yielding an empty
// part. Occurrence placement is the tracked follow-up.
func TestAssemblyIsReportedNotMisTranslated(t *testing.T) {
	out := filepath.Join(t.TempDir(), "asm.opd")
	_, err := FromInventor(readCorpus(t, "asm_two_boxes.iam"), out)
	if err == nil {
		t.Fatal("expected an error translating an assembly as a part, got nil")
	}
	for _, want := range []string{"assembly", "asm_box", "2 occurrence"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("assembly error %q missing %q", err.Error(), want)
		}
	}
}

// TestPartialRevolveTranslationRebuildsSolid translates the 270° revolve (the tube profile
// swept 3/4 of a turn about X) and checks the reopened volume: 3/4 · 4π = 3π cm³. A small
// deficit is curved-surface tessellation.
func TestPartialRevolveTranslationRebuildsSolid(t *testing.T) {
	out := filepath.Join(t.TempDir(), "rev270.opd")
	if _, err := FromInventor(readCorpus(t, "24_revolve_270.ipt"), out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	mp := analysis.MassPropertiesOf(def.SurfaceBodies().All(), 1, types.MassPropertiesHigh)
	want := 3 * math.Pi * 1000 // 3π cm³ in mm³
	if math.Abs(mp.VolumeMm3-want) > 0.02*want {
		t.Errorf("270° revolve volume = %.1f mm^3, want ~%.1f (within 2%%)", mp.VolumeMm3, want)
	}
}

// TestEllipseTranslationRoundTrips translates ke_ellipse.ipt (a two-line base sketch plus one
// ellipse at centre (10,5), major axis +X, majorR 3, minorR 1.5) through the full pipeline and
// checks the reopened .opd carries a native Oblikovati ellipse with those exact parameters — not
// the phantom radius-1 circle it decoded as before the sentinel-gated discriminator.
func TestEllipseTranslationRoundTrips(t *testing.T) {
	def := reopenPart(t, "ke_ellipse.ipt")
	if def.Sketches().Count() != 1 {
		t.Fatalf("got %d sketches, want 1", def.Sketches().Count())
	}
	sk := def.Sketches().Item(0)
	if n := sk.Circles().Count(); n != 0 {
		t.Errorf("got %d circles, want 0 (the ellipse must not emit as a circle)", n)
	}
	if n := sk.Ellipses().Count(); n != 1 {
		t.Fatalf("got %d ellipses, want 1", n)
	}
	e := sk.Ellipses().Item(0)
	if math.Abs(e.MajorRadius-3) > 1e-9 || math.Abs(e.MinorRadius-1.5) > 1e-9 {
		t.Errorf("radii = (%.4g,%.4g), want majorR=3 minorR=1.5", e.MajorRadius, e.MinorRadius)
	}
	if math.Abs(e.Center.X-10) > 1e-9 || math.Abs(e.Center.Y-5) > 1e-9 {
		t.Errorf("centre = (%.4g,%.4g), want (10,5)", e.Center.X, e.Center.Y)
	}
}

// TestSplineTranslationRoundTrips checks the reopened .opd carries a native Oblikovati fit spline
// with the four fit points ke_spline declares — the SketchSpline (0xF9372FD4) decode/emit path.
func TestSplineTranslationRoundTrips(t *testing.T) {
	def := reopenPart(t, "ke_spline.ipt")
	if def.Sketches().Count() != 1 {
		t.Fatalf("got %d sketches, want 1", def.Sketches().Count())
	}
	sk := def.Sketches().Item(0)
	if n := sk.Splines().Count(); n != 1 {
		t.Fatalf("got %d splines, want 1", n)
	}
	sp := sk.Splines().Item(0)
	if !sp.IsFitType() {
		t.Error("spline is not a fit (interpolating) type")
	}
	if n := sp.PointCount(); n != 4 {
		t.Errorf("spline has %d fit points, want 4", n)
	}
}

// TestEmitDroppedCurveSketchesKeepsSplines guards that the freeform-curve rescue (used by the
// revolve/sweep/loft paths, whose line-only profile extraction drops splines and ellipses) emits a
// spline-bearing sketch's spline. Without it a revolve part like Hose-Screen-Adapter loses every
// sketch spline it carries. Uses ke_spline (one fit spline) driven through the rescue directly.
func TestEmitDroppedCurveSketchesKeepsSplines(t *testing.T) {
	d, err := ipt.Open(readCorpus(t, "ke_spline.ipt"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	document, err := compdef.AddPart(ws, filepath.Join(t.TempDir(), "p.opd"), true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := document.Content().(*compdef.PartComponentDefinition)
	emitDroppedCurveSketches(def, d)
	splines := 0
	for k := 0; k < def.Sketches().Count(); k++ {
		splines += def.Sketches().Item(k).Splines().Count()
	}
	if splines != 1 {
		t.Errorf("rescue emitted %d splines, want 1 (the curve a line-only profile decode would drop)", splines)
	}
}

// TestHasBaseExtrude guards the base-detection that keeps a baseless extrude chain (all cut/join,
// no New-Body — MainBaseSheet, whose base plate is a sheet-metal face this decoder doesn't produce)
// from building a garbage sliver: without a New-Body extrude the chain has nothing to cut, so the
// caller imports the real body instead.
func TestHasBaseExtrude(t *testing.T) {
	allCuts := []ipt.Extrude{{Operation: ipt.OpCut}, {Operation: ipt.OpJoin}, {Operation: ipt.OpCut}}
	if hasBaseExtrude(allCuts) {
		t.Error("an all-cut/join chain has no base and must report false")
	}
	withBase := []ipt.Extrude{{Operation: ipt.OpNewBody}, {Operation: ipt.OpCut}}
	if !hasBaseExtrude(withBase) {
		t.Error("a chain containing a New-Body extrude must report true")
	}
	if hasBaseExtrude(nil) {
		t.Error("no extrudes must report false")
	}
}

// reopenPart translates an .ipt through the full pipeline and reopens it, returning the definition.
func reopenPart(t *testing.T, file string) *compdef.PartComponentDefinition {
	t.Helper()
	out := filepath.Join(t.TempDir(), "p.opd")
	if _, err := FromInventor(readCorpus(t, file), out); err != nil {
		t.Fatalf("%s: FromInventor: %v", file, err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("%s: reopen: %v", file, err)
	}
	return reopened.Content().(*compdef.PartComponentDefinition)
}

// TestRealSteppedShaftRevolveRebuildsFullSolid guards two fixes on a real interactively-modelled
// part (ReelMotorBearingShaft): the profile connectivity is reconstructed from point incidence
// (not rank-alignment), and the revolve is a FULL turn. It had regressed to a 1-radian sliver
// (~1910 mm³ = 1/2π of the solid) because a shaft dimension read as a partial sweep angle; the
// correct full solid of the stepped flange profile revolved about x=0 is ~11.9 cm³ (Pappus).
func TestRealSteppedShaftRevolveRebuildsFullSolid(t *testing.T) {
	def := reopenPart(t, "real_shaft_stepped.ipt")
	body := def.SurfaceBodies().All()
	if len(body) == 0 || !body[0].IsSolid() {
		t.Fatal("stepped shaft did not rebuild a parametric solid")
	}
	v := analysis.MassPropertiesOf(body, 1, types.MassPropertiesHigh).VolumeMm3
	if v < 11000 || v > 13000 { // full revolve ≈ 11.9 cm³; a sliver would be ~1910
		t.Errorf("stepped shaft volume = %.0f mm^3, want ~12000 (a sliver ~1910 means the sweep angle regressed)", v)
	}
}

// TestRealSplitClusterShaftRevolveRebuilds guards the cross-cluster reconstruction on TorquimeterShaft:
// its stepped profile is split across the 800-byte cluster gap, so the old rank-alignment left it an
// open chain that declined to the mesh. Incidence reunites the two halves and degree-completes the
// collinear-constrained steps into one closed loop, so the shaft now rebuilds as a parametric solid
// (~0.91 cm³, the faithful revolve of the decoded stepped profile about x=0).
func TestRealSplitClusterShaftRevolveRebuilds(t *testing.T) {
	def := reopenPart(t, "real_shaft_splitcluster.ipt")
	body := def.SurfaceBodies().All()
	if len(body) == 0 || !body[0].IsSolid() {
		t.Fatal("split-cluster shaft did not rebuild a parametric solid (cross-cluster profile not reunited)")
	}
	v := analysis.MassPropertiesOf(body, 1, types.MassPropertiesHigh).VolumeMm3
	if v < 850 || v > 1000 {
		t.Errorf("split-cluster shaft volume = %.0f mm^3, want ~910", v)
	}
}

// TestFullRevolveWithAngleDimStaysFull guards the feature→extent binding: a FULL revolve whose
// profile carries an angle DIMENSION (a chamfer dimensioned to 135°) must sweep the full 360°, not
// mistake that profile angle for the sweep. The feature's kind=12 extent enum (value 3 = full)
// overrides the angle-param scan. Full solid ≈ 12.2 cm³; a 135° sliver would be ~4.6 cm³.
func TestFullRevolveWithAngleDimStaysFull(t *testing.T) {
	def := reopenPart(t, "revolve_full_angledim.ipt")
	body := def.SurfaceBodies().All()
	if len(body) == 0 || !body[0].IsSolid() {
		t.Fatal("full-revolve-with-angle-dim did not rebuild a solid")
	}
	v := analysis.MassPropertiesOf(body, 1, types.MassPropertiesHigh).VolumeMm3
	if v < 11000 || v > 13000 {
		t.Errorf("volume = %.0f mm^3, want ~12200 (a value near 4600 means the 135° profile dim was swept as the angle)", v)
	}
}

// TestPartialRevolveAngleFromByteShape guards the robust angle-value decode on a 150° revolve whose
// angle parameter the "d"-named model-param reader mis-picked (a stray near-zero double read as the
// value → the revolve came out full). soleSweepAngle keys on the value's on-disk shape instead, so
// the 150° sweep rebuilds: 4π·(150/360) ≈ 5.24 cm³ (full would be 4π ≈ 12.57).
func TestPartialRevolveAngleFromByteShape(t *testing.T) {
	def := reopenPart(t, "revolve_partial_150.ipt")
	body := def.SurfaceBodies().All()
	if len(body) == 0 || !body[0].IsSolid() {
		t.Fatal("150° revolve did not rebuild a solid")
	}
	v := analysis.MassPropertiesOf(body, 1, types.MassPropertiesHigh).VolumeMm3
	if want := 4 * math.Pi * 1000 * 150 / 360; math.Abs(v-want) > 0.03*want {
		t.Errorf("150° revolve volume = %.0f mm^3, want ~%.0f (full 4π ≈ 12566 means the partial angle was missed)", v, want)
	}
}

// TestRichHoleTranslationRebuildsSolid translates the counterbore and countersink parts
// (box 4x4x2 = 32 cm^3, Ø0.6 bore through) and checks the reopened volumes match the
// analytic material removed by each recessed hole (exact brep cutters, so tight).
func TestRichHoleTranslationRebuildsSolid(t *testing.T) {
	cases := []struct {
		file    string
		wantMm3 float64
	}{
		{"25_hole_counterbore.ipt", 32000 - (math.Pi*0.7*0.7*0.5+math.Pi*0.3*0.3*1.5)*1000}, // recess + bore
		{"26_hole_countersink.ipt", 31216.7},                                                // cone + bore (from the oracle)
	}
	for _, tc := range cases {
		out := filepath.Join(t.TempDir(), "h.opd")
		if _, err := FromInventor(readCorpus(t, tc.file), out); err != nil {
			t.Fatalf("%s: FromInventor: %v", tc.file, err)
		}
		ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
		reopened, err := ws.Open(out, true)
		if err != nil {
			t.Fatalf("%s: reopen: %v", tc.file, err)
		}
		def := reopened.Content().(*compdef.PartComponentDefinition)
		mp := analysis.MassPropertiesOf(def.SurfaceBodies().All(), 1, types.MassPropertiesHigh)
		if math.Abs(mp.VolumeMm3-tc.wantMm3) > 0.005*tc.wantMm3 {
			t.Errorf("%s: volume = %.1f mm^3, want ~%.1f", tc.file, mp.VolumeMm3, tc.wantMm3)
		}
	}
}

// TestTappedHoleTranslation translates the tapped M6x1 hole and checks the reopened part:
// the bore is a plain tap-drill cylinder (box 4x4x2 minus Ø0.4917 through ≈ 31.62 cm³) and
// the hole carries its thread designation through the recipe round-trip.
func TestTappedHoleTranslation(t *testing.T) {
	out := filepath.Join(t.TempDir(), "tap.opd")
	if _, err := FromInventor(readCorpus(t, "27_hole_tapped.ipt"), out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	mp := analysis.MassPropertiesOf(def.SurfaceBodies().All(), 1, types.MassPropertiesHigh)
	if want := 31624.5; math.Abs(mp.VolumeMm3-want) > 0.01*want {
		t.Errorf("tapped-hole volume = %.1f mm^3, want ~%.1f", mp.VolumeMm3, want)
	}
	var designation string
	for i := 0; i < def.Features().Count(); i++ {
		if hf, ok := def.Features().Item(i).Definition().(*feature.HoleFeature); ok {
			designation = hf.Definition().Tap.Designation
		}
	}
	if designation != "M6x1" {
		t.Errorf("reopened hole thread designation = %q, want M6x1 (did not round-trip)", designation)
	}
}

// TestLoftTranslationRebuildsSolid translates the loft (a 2x2 square at z=0 blended to a 4x4
// square at z=4) parametrically and checks the reopened volume: the ruled loft is a square
// frustum, ∫₀⁴ (2 + z/2)² dz = 112/3 ≈ 37.33 cm³.
func TestLoftTranslationRebuildsSolid(t *testing.T) {
	out := filepath.Join(t.TempDir(), "loft.opd")
	if _, err := FromInventor(readCorpus(t, "28_loft.ipt"), out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	if def.Features().Count() == 0 {
		t.Fatalf("reopened loft has no features (not rebuilt parametrically)")
	}
	mp := analysis.MassPropertiesOf(def.SurfaceBodies().All(), 1, types.MassPropertiesHigh)
	if want := 112.0 / 3 * 1000; math.Abs(mp.VolumeMm3-want) > 1 {
		t.Errorf("loft volume = %.1f mm^3, want %.1f (112/3)", mp.VolumeMm3, want)
	}
}

// TestSweepTranslationRebuildsSolid translates the sweep (a Ø0.5 circle along an L-path, two
// 5 cm legs) and checks the reopened body is a valid elbow of the right order: well above a
// single 5 cm leg (π·0.25·5 ≈ 3.9 cm³), so both legs swept. The exact volume differs from
// Inventor's oracle (7.8 cm³) because the two kernels treat the sharp path corner
// differently — Inventor bends smoothly, Oblikovati miters.
func TestSweepTranslationRebuildsSolid(t *testing.T) {
	out := filepath.Join(t.TempDir(), "sweep.opd")
	if _, err := FromInventor(readCorpus(t, "29_sweep.ipt"), out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	if def.Features().Count() == 0 {
		t.Fatalf("reopened sweep has no features (not rebuilt parametrically)")
	}
	mp := analysis.MassPropertiesOf(def.SurfaceBodies().All(), 1, types.MassPropertiesHigh)
	if mp.VolumeMm3 < 6000 || mp.VolumeMm3 > 8200 {
		t.Errorf("swept elbow volume = %.1f mm^3, want a full two-leg elbow (~6600–7900)", mp.VolumeMm3)
	}
}

// TestArcSweepTranslationRebuildsSolid translates the arc-path sweep (a Ø0.5 circle along a
// quarter-circle arc of radius 3) and checks the reopened volume ≈ the tube volume
// π·0.25·(π·3/2) ≈ 3.70 cm³. The arc is smooth (no sharp corner), so it matches Inventor's
// oracle closely (small deficit is arc + curved-surface tessellation).
func TestArcSweepTranslationRebuildsSolid(t *testing.T) {
	out := filepath.Join(t.TempDir(), "arc.opd")
	if _, err := FromInventor(readCorpus(t, "30_sweep_arc.ipt"), out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	mp := analysis.MassPropertiesOf(def.SurfaceBodies().All(), 1, types.MassPropertiesHigh)
	if want := math.Pi * 0.25 * (math.Pi * 3 / 2) * 1000; math.Abs(mp.VolumeMm3-want) > 0.05*want {
		t.Errorf("arc-sweep volume = %.1f mm^3, want ~%.1f (within 5%%)", mp.VolumeMm3, want)
	}
}

// TestMultiSectionLoftTranslationRebuildsSolid translates the 3-section loft (2x2 at z=0,
// 4x4 at z=3, 1x1 at z=6) and checks the reopened body blends through all three sections at
// their heights: 3 sketches, and a volume well above the ~14 cm³ a loft skipping the middle
// 4x4 section would give. The exact volume differs from Inventor's oracle (63.9 cm³) — the
// two kernels bulge differently through interior sections (Oblikovati ≈ 57.2 cm³).
func TestMultiSectionLoftTranslationRebuildsSolid(t *testing.T) {
	out := filepath.Join(t.TempDir(), "loft3.opd")
	if _, err := FromInventor(readCorpus(t, "31_loft3.ipt"), out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	if def.Sketches().Count() != 3 {
		t.Errorf("reopened loft has %d sketches, want 3 sections", def.Sketches().Count())
	}
	mp := analysis.MassPropertiesOf(def.SurfaceBodies().All(), 1, types.MassPropertiesHigh)
	if mp.VolumeMm3 < 50000 || mp.VolumeMm3 > 66000 {
		t.Errorf("3-section loft volume = %.1f mm^3, want a full three-section blend (~57000–64000)", mp.VolumeMm3)
	}
}

// TestMirrorTranslationRebuildsSolid translates the pocket + mirror part: a 6x4x1 = 24 cm^3
// box with a 0.6x0.6 through-pocket at (1,1), mirrored across the x=3 plane to (5,1). Two
// pockets remove 2*(0.6*0.6*1) = 0.72 cm^3 → 23.28. Proves the mirror plane + reflection.
func TestMirrorTranslationRebuildsSolid(t *testing.T) {
	out := filepath.Join(t.TempDir(), "mir.opd")
	if _, err := FromInventor(readCorpus(t, "23_pocket_mirror.ipt"), out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	if def.Features().Count() < 3 {
		t.Fatalf("reopened part has %d features, want >=3 (box + pocket + mirror)", def.Features().Count())
	}
	mp := analysis.MassPropertiesOf(def.SurfaceBodies().All(), 1, types.MassPropertiesHigh)
	if want := 24000.0 - 2*0.36*1000; math.Abs(mp.VolumeMm3-want) > 1 {
		t.Errorf("mirrored volume = %.1f mm^3, want %.1f", mp.VolumeMm3, want)
	}
}

// TestLinkageProfileMatchesTheFile pins the real linkage's sketch to what the FILE actually holds.
//
// The fixture is ReelToReel's CompressionRollerActuatorLinkage1 (true volume 45 mm^3, Inventor
// 2027). Its sketch contains exactly 4 circles and 3 lines and ZERO arcs — its rounded ends are
// full circles trimmed by the boundary patch. The predecessor of this test asserted that arcs must
// be emitted; that expectation came from the old cluster decode, which FABRICATED arcs this file
// does not contain. Like the "want ~477 mm^3" before it, it codified our own output as truth.
// GraphSketches now decodes the entities the file declares, so the counts below come from the file.
func TestLinkageProfileMatchesTheFile(t *testing.T) {
	def := reopenPart(t, "real_arc_linkage.ipt")
	if n := def.Sketches().Count(); n != 1 {
		t.Errorf("emitted %d sketches; the file declares ONE Sketch2D node", n)
	}
	circles, lines, arcs := 0, 0, 0
	for k := 0; k < def.Sketches().Count(); k++ {
		s := def.Sketches().Item(k)
		circles += s.Circles().Count()
		lines += s.Lines().Count()
		arcs += s.Arcs().Count()
	}
	if circles != 4 || lines != 3 || arcs != 0 {
		t.Errorf("sketch has %d circles / %d lines / %d arcs; the file declares 4 / 3 / 0",
			circles, lines, arcs)
	}
	bodies := def.SurfaceBodies().All()
	if len(bodies) == 0 {
		return // no body shipped is acceptable; a wrong one is not
	}
	// A regression guard, NOT a claim that the current volume is right: the extrude still takes
	// profile index 0 rather than the region its BoundaryPatch names, so the body is undersized.
	// This catches a return of the old 477 mm^3 (10.5x) over-build from the positional depth guess.
	if v := analysis.MassPropertiesOf(bodies, 1, types.MassPropertiesHigh).VolumeMm3; v > 100 {
		t.Errorf("linkage volume = %.0f mm^3; true volume is 45 mm^3 — a >2x over-build means the "+
			"extrude is taking a wrong depth again", v)
	}
}

// TestProfileCornersShareCoincidentPoints guards that a rebuilt profile's touching corners are
// ONE shared sketch point, not independent duplicated endpoints — reproducing the original's
// endpoint coincidence constraints and so its degrees of freedom. The L-profile is a closed
// 6-corner loop: 6 shared points ⇒ 12 free DOF (2 per point), not 24 (4 per unconstrained line).
func TestProfileCornersShareCoincidentPoints(t *testing.T) {
	def := reopenPart(t, "18_lprofile.ipt")
	s := def.Sketches().Item(0)
	if got := s.Points().Count(); got != 6 {
		t.Errorf("L-profile sketch has %d points, want 6 (corners must share one coincident point)", got)
	}
	if got := s.Lines().Count(); got != 6 {
		t.Errorf("L-profile sketch has %d lines, want 6", got)
	}
	if dof := s.DegreesOfFreedom(); dof != 12 {
		t.Errorf("L-profile sketch DOF = %d, want 12 (a coincident 6-gon; 24 means corners were left independent)", dof)
	}
}

// TestRevolveCentrelineReunited guards that a revolve's separate vertical centreline is merged
// back into its profile sketch: the stepped shaft, whose incidence decode splits the centreline
// into its own component, must rebuild as ONE sketch containing the 6 profile edges plus the
// vertical x≈0 centreline (7 lines) — so the revolve's radius dimensions can bind in one sketch —
// and the revolve solid must be unchanged (same axis line, ~11.9 cm³).
func TestRevolveCentrelineReunited(t *testing.T) {
	def := reopenPart(t, "real_shaft_stepped.ipt")
	if n := def.Sketches().Count(); n != 1 {
		t.Errorf("stepped shaft has %d sketches, want 1 (centreline reunited into the profile)", n)
	}
	s := def.Sketches().Item(0)
	if s.Lines().Count() != 7 {
		t.Errorf("reunited sketch has %d lines, want 7 (6 profile + centreline)", s.Lines().Count())
	}
	axis := 0
	for i := 0; i < s.Lines().Count(); i++ {
		l := s.Lines().Item(i)
		if math.Abs(float64(l.A.X)) < 1e-4 && math.Abs(float64(l.B.X)) < 1e-4 {
			axis++
		}
	}
	if axis != 1 {
		t.Errorf("reunited sketch has %d lines on x=0, want 1 (the centreline)", axis)
	}
	body := def.SurfaceBodies().All()
	if len(body) == 0 || !body[0].IsSolid() {
		t.Fatal("shaft no longer rebuilds a solid after reuniting the centreline")
	}
	if v := analysis.MassPropertiesOf(body, 1, types.MassPropertiesHigh).VolumeMm3; v < 11000 || v > 13000 {
		t.Errorf("shaft volume = %.0f mm^3, want ~12000 — reuniting the centreline changed the solid", v)
	}
}

// TestGeometricConstraintsReduceDOF guards that decoded geometric constraints (parallel /
// perpendicular / horizontal / vertical) are applied to the rebuilt sketch and remove degrees of
// freedom — the real-part step toward DOF parity. The stepped-shaft profile carries perpendicular
// constraints between its steps; applying them drops its 6-corner profile below the coincidence-only
// 12 DOF. Its revolve solid must still rebuild unchanged (the constraints are already satisfied, so
// they pin DOF without moving geometry).
func TestGeometricConstraintsReduceDOF(t *testing.T) {
	def := reopenPart(t, "real_shaft_stepped.ipt")
	profile := profileSketch(def)
	if profile == nil {
		t.Fatal("no 6-line profile sketch found")
	}
	a := profile.AnalyzeConstraints()
	if a.Equations == 0 {
		t.Error("shaft profile has no constraint equations — geometric constraints were not applied")
	}
	if a.DOF >= 12 {
		t.Errorf("shaft profile DOF = %d, want < 12 (constraints must reduce it below the coincidence-only baseline)", a.DOF)
	}
	// geometry must be unchanged: the revolve solid still rebuilds to ~11.9 cm³ (Pappus).
	body := def.SurfaceBodies().All()
	if len(body) == 0 || !body[0].IsSolid() {
		t.Fatal("shaft no longer rebuilds a solid after applying constraints")
	}
	if v := analysis.MassPropertiesOf(body, 1, types.MassPropertiesHigh).VolumeMm3; v < 11000 || v > 13000 {
		t.Errorf("shaft volume = %.0f mm^3, want ~12000 — constraints moved geometry", v)
	}
}

// TestShaftConstraintPipeline guards the cumulative DOF-parity pipeline on the stepped shaft:
// shared coincident corners + perpendicular constraints + radius dimensions + axial step-length
// dimensions + the centreline origin anchor bring its free 24-DOF profile down to a well-driven
// sketch (≤6 DOF, no redundant constraints), while the revolve solid stays exact (~11.9 cm³).
func TestShaftConstraintPipeline(t *testing.T) {
	def := reopenPart(t, "real_shaft_stepped.ipt")
	s := profileSketch(def)
	if s == nil {
		t.Fatal("no revolve profile sketch")
	}
	a := s.AnalyzeConstraints()
	if a.DOF > 6 {
		t.Errorf("shaft profile DOF = %d, want ≤ 6 (constraints + dimensions + anchor should drive it down from 24)", a.DOF)
	}
	if a.Redundant != 0 {
		t.Errorf("shaft profile has %d redundant constraints, want 0 (no over-constraint)", a.Redundant)
	}
	body := def.SurfaceBodies().All()
	if v := analysis.MassPropertiesOf(body, 1, types.MassPropertiesHigh).VolumeMm3; v < 11000 || v > 13000 {
		t.Errorf("shaft volume = %.0f mm^3, want ~12000 — a constraint moved geometry", v)
	}
}

// profileSketch returns the part's revolve profile sketch — the stepped shaft's 6 profile edges
// plus the reunited vertical centreline (7 lines).
func profileSketch(def *compdef.PartComponentDefinition) *sketch.Sketch {
	for k := 0; k < def.Sketches().Count(); k++ {
		if s := def.Sketches().Item(k); s.Lines().Count() >= 6 {
			return s
		}
	}
	return nil
}

// TestSketchExtractionDecoupledFromFeatures guards the decoupling of sketch extraction from
// feature building: a sketch-only part (no feature) still emits its sketch, and — with the mesh
// fallback OFF by default — no opaque display body is imported to mask it. This is the property
// that makes a partially-translated part inspectable in the browser.
func TestSketchExtractionDecoupledFromFeatures(t *testing.T) {
	def := reopenPart(t, "sketch_line.ipt")
	if n := def.Sketches().Count(); n != 1 {
		t.Errorf("reopened part has %d sketches, want 1 (sketch not emitted independently of a feature)", n)
	}
	if bodies := def.SurfaceBodies().All(); len(bodies) != 0 {
		t.Errorf("reopened sketch-only part has %d bodies, want 0 (a mesh body was imported by default, hiding the partial state)", len(bodies))
	}
}

// TestMeshFallbackGateHonorsEnv guards the toggle that chooses partial-parametric (default) vs
// the opaque display mesh. The default is OFF so troubleshooting sees the real tree;
// OBK_IPT_MESH_FALLBACK=1 turns it back on.
func TestMeshFallbackGateHonorsEnv(t *testing.T) {
	t.Setenv("OBK_IPT_MESH_FALLBACK", "")
	if meshFallbackEnabled() {
		t.Error("mesh fallback should be OFF by default")
	}
	t.Setenv("OBK_IPT_MESH_FALLBACK", "1")
	if !meshFallbackEnabled() {
		t.Error("OBK_IPT_MESH_FALLBACK=1 should enable the mesh fallback")
	}
}

// TestImportRoundTripsThroughOPD imports the box to a .opd, reopens it with a fresh
// workspace, and checks the reconstructed-then-reloaded body still measures 8 cm^3 — proving
// the body persists through the recipe and re-derives on open.
func TestImportRoundTripsThroughOPD(t *testing.T) {
	out := filepath.Join(t.TempDir(), "box.opd")
	if _, err := FromInventor(readCorpus(t, "10_box.ipt"), out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen %s: %v", out, err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	bodies := def.SurfaceBodies().All()
	if len(bodies) != 1 {
		t.Fatalf("reopened part has %d bodies, want 1", len(bodies))
	}
	mp := analysis.MassPropertiesOf([]*topo.Body{bodies[0]}, 1, types.MassPropertiesHigh)
	if math.Abs(mp.VolumeMm3-8000) > 1 {
		t.Errorf("reopened box volume = %.3f mm^3, want 8000", mp.VolumeMm3)
	}
}

// TestCapstainNutBoreIsDrilledOnItsOwnFace pins the hole-placement fix: CapstainNut's hex is
// extruded along +X, so addHole's top-face fallback (XY-plane assumption) drilled in empty space and
// the Ø1.7 bore removed nothing (a solid nut, 3663 mm³ = 1.62x). Using the hole's own placement
// (its transform's point + axis) as the GeomFace AND the drill Center bores it correctly — 2420 mm³
// vs Inventor's 2261 (1.07x; the residual is the two 45° chamfers this decoder does not build).
// Corpus-gated: the non-XY-face bore only exists in the real part; skips without IPT_CORPUS.
func TestCapstainNutBoreIsDrilledOnItsOwnFace(t *testing.T) {
	dir := os.Getenv("IPT_CORPUS")
	if dir == "" {
		dir = `P:\ReelToReel\Mechanical`
	}
	data, err := os.ReadFile(filepath.Join(dir, "CapstainNut.ipt"))
	if err != nil {
		t.Skipf("corpus part not available (%v); set IPT_CORPUS", err)
	}
	out := filepath.Join(t.TempDir(), "nut.opd")
	if _, err := FromInventor(data, out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	bodies := def.SurfaceBodies().All()
	if len(bodies) == 0 {
		t.Fatalf("no body built")
	}
	vol := analysis.MassPropertiesOf(bodies, 1, types.MassPropertiesLow).VolumeMm3
	const oracle = 2261.0 // Inventor STL volume, mm³
	if vol > 1.2*oracle {
		t.Errorf("CapstainNut volume = %.0f mm³, want <= %.0f (the bore must be drilled, not missing)", vol, 1.2*oracle)
	}
	if math.Abs(vol-oracle) > 0.12*oracle {
		t.Errorf("CapstainNut volume = %.0f mm³, want within 12%% of Inventor's %.0f", vol, oracle)
	}
}

// TestFlangeReelMotorCutFitsItsScallops pins the containment area-fit guard. FlangeReelMotor's third
// extrude is a through-all cut whose region is four ~13.7 cm² edge scallops. The +-shaped keep cell
// (108 cm²) got a test point inside one scallop loop, so containment selected the whole + and the
// through-cut gutted the flange (19468 mm³ = 0.11x). A cell far larger than the loop holding its
// point cannot be that loop's interior, so it is now rejected; only the scallops are cut and the
// flange survives (183104 vs Inventor's 175398 = 1.04x). Corpus-gated; set IPT_CORPUS.
func TestFlangeReelMotorCutFitsItsScallops(t *testing.T) {
	dir := os.Getenv("IPT_CORPUS")
	if dir == "" {
		dir = `P:\ReelToReel\Mechanical`
	}
	data, err := os.ReadFile(filepath.Join(dir, "FlangeReelMotor.ipt"))
	if err != nil {
		t.Skipf("corpus part not available (%v); set IPT_CORPUS", err)
	}
	out := filepath.Join(t.TempDir(), "flange.opd")
	if _, err := FromInventor(data, out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	bodies := def.SurfaceBodies().All()
	if len(bodies) == 0 {
		t.Fatalf("no body built")
	}
	vol := analysis.MassPropertiesOf(bodies, 1, types.MassPropertiesLow).VolumeMm3
	const oracle = 175398.0 // Inventor STL volume, mm³
	// Guard against a regression back to the gutted flange (19468 = 0.11x).
	if vol < 0.5*oracle {
		t.Errorf("FlangeReelMotor volume = %.0f mm³, want >= %.0f (the cut must not gut the flange)", vol, 0.5*oracle)
	}
	if math.Abs(vol-oracle) > 0.08*oracle {
		t.Errorf("FlangeReelMotor volume = %.0f mm³, want within 8%% of Inventor's %.0f", vol, oracle)
	}
}

// TestHorizontalAxisRevolveBuildsSolids pins the horizontal-centreline revolve. A shaft/bushing
// turned about a HORIZONTAL axis (an isolated line running along X, e.g. 1677K262's line at y=0)
// was not recognised — revolveAxisIndex only accepted a vertical centreline — so the part fell back
// to a 2x open-mesh SURFACE. Now an axis-aligned isolated line is a valid centreline and the profile
// turns a full 360° (the partial-angle decode is unreliable about a horizontal axis). Corpus-gated;
// set IPT_CORPUS.
func TestHorizontalAxisRevolveBuildsSolids(t *testing.T) {
	dir := os.Getenv("IPT_CORPUS")
	if dir == "" {
		dir = `P:\ReelToReel\Mechanical`
	}
	for _, tc := range []struct {
		file   string
		oracle float64
	}{
		{"1677K262.ipt", 2586},            // a turned bushing, full 360° about y=0
		{"CapstainFrontBody.ipt", 114417}, // full turn about a horizontal axis (its 125° is a chamfer, not the sweep)
	} {
		data, err := os.ReadFile(filepath.Join(dir, tc.file))
		if err != nil {
			t.Skipf("corpus part not available (%v); set IPT_CORPUS", err)
		}
		out := filepath.Join(t.TempDir(), "r.opd")
		if _, err := FromInventor(data, out); err != nil {
			t.Fatalf("%s: FromInventor: %v", tc.file, err)
		}
		ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
		reopened, err := ws.Open(out, true)
		if err != nil {
			t.Fatalf("%s: reopen: %v", tc.file, err)
		}
		def := reopened.Content().(*compdef.PartComponentDefinition)
		bodies := def.SurfaceBodies().All()
		if len(bodies) == 0 || !bodies[0].IsSolid() {
			t.Fatalf("%s: want a SOLID revolve, got %d bodies (solid=%v)", tc.file, len(bodies), len(bodies) > 0 && bodies[0].IsSolid())
		}
		vol := analysis.MassPropertiesOf(bodies, 1, types.MassPropertiesLow).VolumeMm3
		if math.Abs(vol-tc.oracle) > 0.05*tc.oracle {
			t.Errorf("%s: volume = %.0f mm³, want within 5%% of Inventor's %.0f", tc.file, vol, tc.oracle)
		}
	}
}

// TestMachinedHolderRevolveWithCuts covers the ext,rev,hole path (buildRevolveDispatch's graph
// branch): a turned base whose milled cut extrudes must be applied over it. The incidence line set
// splits the revolve profile from its centreline, so the base is rebuilt from the node graph (which
// keeps them whole) and the cuts are booleaned on top. Both parts reopen as a SOLID within 2% of
// Inventor's STL volume (SpoolMotor 19441, CapstonMotor 17201). Corpus-gated; set IPT_CORPUS.
func TestMachinedHolderRevolveWithCuts(t *testing.T) {
	dir := os.Getenv("IPT_CORPUS")
	if dir == "" {
		dir = `P:\ReelToReel\Mechanical`
	}
	for _, tc := range []struct {
		file   string
		oracle float64
	}{
		{"SpoolMotorMachinedHolder.ipt", 19441},
		{"CapstonMotorMachinedHolder.ipt", 17201},
	} {
		data, err := os.ReadFile(filepath.Join(dir, tc.file))
		if err != nil {
			t.Skipf("corpus part not available (%v); set IPT_CORPUS", err)
		}
		out := filepath.Join(t.TempDir(), "h.opd")
		if _, err := FromInventor(data, out); err != nil {
			t.Fatalf("%s: FromInventor: %v", tc.file, err)
		}
		ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
		reopened, err := ws.Open(out, true)
		if err != nil {
			t.Fatalf("%s: reopen: %v", tc.file, err)
		}
		def := reopened.Content().(*compdef.PartComponentDefinition)
		bodies := def.SurfaceBodies().All()
		if len(bodies) == 0 || !bodies[0].IsSolid() {
			t.Fatalf("%s: want a SOLID turned+milled body, got %d bodies (solid=%v)", tc.file, len(bodies), len(bodies) > 0 && bodies[0].IsSolid())
		}
		vol := analysis.MassPropertiesOf(bodies, 1, types.MassPropertiesLow).VolumeMm3
		if math.Abs(vol-tc.oracle) > 0.02*tc.oracle {
			t.Errorf("%s: volume = %.0f mm³, want within 2%% of Inventor's %.0f", tc.file, vol, tc.oracle)
		}
	}
}

// TestCapstainMotorCapRevolvesAboutItsEdge covers the axis-reference fallback (Phase 2): the revolve
// turns about its y=0 top EDGE, which is neither isolated nor construction, so the geometric
// heuristic returns nothing and the feature's decoded axis (ipt.RevolveAxis2D) supplies it. Reopens
// as a SOLID within 3% of Inventor's STL volume (31223). Corpus-gated; set IPT_CORPUS.
func TestCapstainMotorCapRevolvesAboutItsEdge(t *testing.T) {
	dir := os.Getenv("IPT_CORPUS")
	if dir == "" {
		dir = `P:\ReelToReel\Mechanical`
	}
	data, err := os.ReadFile(filepath.Join(dir, "CapstainMotorCap.ipt"))
	if err != nil {
		t.Skipf("corpus part not available (%v); set IPT_CORPUS", err)
	}
	out := filepath.Join(t.TempDir(), "cmc.opd")
	if _, err := FromInventor(data, out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	bodies := def.SurfaceBodies().All()
	if len(bodies) == 0 || !bodies[0].IsSolid() {
		t.Fatalf("want a SOLID turned+milled cap, got %d bodies (solid=%v)", len(bodies), len(bodies) > 0 && bodies[0].IsSolid())
	}
	vol := analysis.MassPropertiesOf(bodies, 1, types.MassPropertiesLow).VolumeMm3
	if math.Abs(vol-31223) > 0.03*31223 {
		t.Errorf("volume = %.0f mm³, want within 3%% of Inventor's 31223", vol)
	}
}

// TestSmartKnobFixedBaseCutsThroughStepped covers the through-cut retry: a blind/one-sided cut on
// this turned knob lands its end face coincident with the stepped top and OPENS the body; applyRevolveCuts
// retries such a cut as a symmetric through-all, which removes the full column and closes it. Reopens
// as a SOLID within 2% of Inventor's STL volume (3549). Corpus-gated; set IPT_CORPUS.
func TestSmartKnobFixedBaseCutsThroughStepped(t *testing.T) {
	dir := os.Getenv("IPT_CORPUS")
	if dir == "" {
		dir = `P:\ReelToReel\Mechanical`
	}
	data, err := os.ReadFile(filepath.Join(dir, "SmartKnobFixedBase.ipt"))
	if err != nil {
		t.Skipf("corpus part not available (%v); set IPT_CORPUS", err)
	}
	out := filepath.Join(t.TempDir(), "skfb.opd")
	if _, err := FromInventor(data, out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	bodies := def.SurfaceBodies().All()
	if len(bodies) == 0 || !bodies[0].IsSolid() {
		t.Fatalf("want a SOLID turned+milled base, got %d bodies (solid=%v)", len(bodies), len(bodies) > 0 && bodies[0].IsSolid())
	}
	vol := analysis.MassPropertiesOf(bodies, 1, types.MassPropertiesLow).VolumeMm3
	if math.Abs(vol-3549) > 0.02*3549 {
		t.Errorf("volume = %.0f mm³, want within 2%% of Inventor's 3549", vol)
	}
}
