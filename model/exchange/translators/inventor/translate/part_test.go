// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"math"
	"path/filepath"
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/analysis"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
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

// TestArcProfileRebuildsSolid guards arc emission: a real filleted linkage part whose profile mixes
// lines and arcs must now emit its arcs (so the profile closes) and extrude to a solid — before arc
// emission its open profile computed to no body (PARTIAL). The arc endpoints are shared with the
// adjacent lines, so the loop is watertight.
func TestArcProfileRebuildsSolid(t *testing.T) {
	def := reopenPart(t, "real_arc_linkage.ipt")
	arcs := 0
	for k := 0; k < def.Sketches().Count(); k++ {
		arcs += def.Sketches().Item(k).Arcs().Count()
	}
	if arcs == 0 {
		t.Error("filleted profile emitted no arcs (arc emission missing)")
	}
	body := def.SurfaceBodies().All()
	if len(body) == 0 || !body[0].IsSolid() {
		t.Fatal("arc-profile linkage did not rebuild a solid (open profile)")
	}
	if v := analysis.MassPropertiesOf(body, 1, types.MassPropertiesHigh).VolumeMm3; v < 400 || v > 560 {
		t.Errorf("arc-profile linkage volume = %.0f mm^3, want ~477 (a wrong-direction arc would bulge it)", v)
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
