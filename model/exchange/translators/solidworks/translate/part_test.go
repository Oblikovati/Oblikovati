// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/persistence"
)

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "testdata", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return b
}

// reopen translates src to a temp .opd and reopens it as an Oblikovati part definition.
func reopen(t *testing.T, src string) *compdef.PartComponentDefinition {
	t.Helper()
	out := filepath.Join(t.TempDir(), "out.opd")
	if _, err := FromSolidWorks(readTestdata(t, src), out); err != nil {
		t.Fatalf("%s: FromSolidWorks: %v", src, err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	document, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("%s: reopen: %v", src, err)
	}
	return document.Content().(*compdef.PartComponentDefinition)
}

// TestParameterTranslation translates the global variables and checks the reopened part carries
// them as native parameters in database units: width 20mm -> 2cm, height 12mm -> 1.2cm, count 5.
func TestParameterTranslation(t *testing.T) {
	def := reopen(t, "params_fmtb.sldprt")
	want := map[string]float64{"width": 2.0, "height": 1.2, "count": 5.0}
	for name, cm := range want {
		p, ok := def.Parameters().ByName(name)
		if !ok {
			t.Errorf("reopened part has no parameter %q", name)
			continue
		}
		if math.Abs(p.NominalValue()-cm) > 1e-9 {
			t.Errorf("parameter %s = %.4f, want %.4f (cm)", name, p.NominalValue(), cm)
		}
	}
}

// TestSketchTranslation translates a rectangle and a circle and checks the reopened part carries
// one sketch each, converted to centimetres (a 10mm box -> a 1cm square).
func TestSketchTranslation(t *testing.T) {
	cases := []struct {
		file           string
		lines, circles int
	}{
		{"box10_fmtb.sldprt", 4, 0},
		{"cyl_fmtb.sldprt", 0, 1},
	}
	for _, c := range cases {
		def := reopen(t, c.file)
		if n := def.Sketches().Count(); n != 1 {
			t.Fatalf("%s: %d sketches, want 1", c.file, n)
		}
		sk := def.Sketches().Item(0)
		if got := sk.Lines().Count(); got != c.lines {
			t.Errorf("%s: %d lines, want %d", c.file, got, c.lines)
		}
		if got := sk.Circles().Count(); got != c.circles {
			t.Errorf("%s: %d circles, want %d", c.file, got, c.circles)
		}
	}
}

// TestConstraintApplication checks that the decoded relations are applied to the emitted geometry,
// removing degrees of freedom: a square rectangle (2 horizontal + 2 vertical + equal-length) lands
// at 3 DOF (position + size) instead of a free 8, and a triangle (1 horizontal + 1 vertical) at 4
// instead of 6. A single circle has exactly 3 DOF (centre + radius) with no stray free points.
func TestConstraintApplication(t *testing.T) {
	cases := []struct {
		file string
		dof  int
	}{
		{"box10_fmtb.sldprt", 3},
		{"tri_fmtb.sldprt", 4},
		{"cyl_fmtb.sldprt", 3},
	}
	for _, c := range cases {
		def := reopen(t, c.file)
		if got := def.Sketches().Item(0).DegreesOfFreedom(); got != c.dof {
			t.Errorf("%s: sketch DOF = %d, want %d", c.file, got, c.dof)
		}
	}
}

// TestDimensionApplication checks that decoded dimensions are applied, pinning the geometry they
// measure: a 25x15 mm dimensioned rectangle drops to 2 DOF (only its position is free — width and
// height are set), and a Ø8 mm dimensioned circle to 2 DOF (only its centre is free — the diameter
// is set). The value-matching infers the kind (a line length -> distance, 2x radius -> diameter).
func TestDimensionApplication(t *testing.T) {
	for _, c := range []struct {
		file string
		dof  int
	}{
		{"dimrect_fmtb.sldprt", 2},
		{"dimcirc_fmtb.sldprt", 2},
	} {
		def := reopen(t, c.file)
		if got := def.Sketches().Item(0).DegreesOfFreedom(); got != c.dof {
			t.Errorf("%s: sketch DOF = %d, want %d", c.file, got, c.dof)
		}
	}
}

// TestConstructionAttribution checks that a construction (reference) entity survives translation and
// reopening: a two-circle sketch whose second circle was drawn as construction geometry reopens with
// exactly one of its two circles marked construction, in the source draw order (real, then
// construction). Construction geometry is excluded from profiles, so preserving it keeps extrudes
// correct.
func TestConstructionAttribution(t *testing.T) {
	def := reopen(t, "constrcirc_fmtb.sldprt")
	sk := def.Sketches().Item(0)
	if sk.Circles().Count() != 2 {
		t.Fatalf("got %d circles, want 2", sk.Circles().Count())
	}
	want := []bool{false, true}
	for i, w := range want {
		if got := sk.Circles().Item(i).IsConstruction(); got != w {
			t.Errorf("circle %d IsConstruction = %v, want %v", i, got, w)
		}
	}
}

// TestLineConstructionAttribution checks construction lines survive translation: sketches with one
// and with two construction lines (the latter interleaved with real lines in draw order) reopen with
// the right number of lines marked construction, so they are excluded from profiles.
func TestLineConstructionAttribution(t *testing.T) {
	for _, c := range []struct {
		file          string
		lines, constr int
	}{
		{"constrline_fmtb.sldprt", 2, 1},
		{"mixconstr_fmtb.sldprt", 4, 2}, // 2 real + 2 construction, interleaved
	} {
		def := reopen(t, c.file)
		sk := def.Sketches().Item(0)
		if sk.Lines().Count() != c.lines {
			t.Fatalf("%s: got %d lines, want %d", c.file, sk.Lines().Count(), c.lines)
		}
		constr := 0
		for i := 0; i < sk.Lines().Count(); i++ {
			if sk.Lines().Item(i).IsConstruction() {
				constr++
			}
		}
		if constr != c.constr {
			t.Errorf("%s: got %d construction lines, want %d", c.file, constr, c.constr)
		}
	}
}

// TestOpenProfileTranslation checks the open-profile fix survives translation: an angle sketch of two
// lines meeting at the origin reopens as two lines (an open V), not the spurious closed triangle the
// old convex-loop reconstruction produced. The exact entity-reference topology carries through.
func TestOpenProfileTranslation(t *testing.T) {
	def := reopen(t, "angledim_fmtb.sldprt")
	sk := def.Sketches().Item(0)
	if got := sk.Lines().Count(); got != 2 {
		t.Errorf("open angle sketch reopened with %d lines, want 2 (not a closed triangle)", got)
	}
}

// TestPointPairDistance checks a distance dimension between two sketch points is applied: two free
// points (4 DOF) with a 30 mm distance between them drop to 3 DOF (the separation is pinned).
func TestPointPairDistance(t *testing.T) {
	def := reopen(t, "ptdist_fmtb.sldprt")
	sk := def.Sketches().Item(0)
	if got := sk.DegreesOfFreedom(); got != 3 {
		t.Errorf("sketch DOF = %d, want 3 (two points with a distance dimension)", got)
	}
}

// TestEllipseTranslation checks an ellipse survives translation to the native part: a 30x20 mm
// ellipse reopens as one ellipse with the semi-axes converted to centimetres (3 and 2 cm).
func TestEllipseTranslation(t *testing.T) {
	def := reopen(t, "ellipse_fmtb.sldprt")
	sk := def.Sketches().Item(0)
	if sk.Ellipses().Count() != 1 {
		t.Fatalf("got %d ellipses, want 1", sk.Ellipses().Count())
	}
	e := sk.Ellipses().Item(0)
	if math.Abs(float64(e.MajorRadius)-3) > 1e-6 || math.Abs(float64(e.MinorRadius)-2) > 1e-6 {
		t.Errorf("radii = %g/%g cm, want 3/2", float64(e.MajorRadius), float64(e.MinorRadius))
	}
}

// TestSplineTranslation checks a fit-point spline survives translation: a four-point spline reopens
// as one spline with four fit points.
func TestSplineTranslation(t *testing.T) {
	def := reopen(t, "spline_fmtb.sldprt")
	sk := def.Sketches().Item(0)
	if sk.Splines().Count() != 1 {
		t.Fatalf("got %d splines, want 1", sk.Splines().Count())
	}
}

// TestMixedSketchTranslation translates the rounded rectangle (4 lines + 4 fillet arcs) and checks
// both kinds persist.
func TestMixedSketchTranslation(t *testing.T) {
	def := reopen(t, "rrect_fmtb.sldprt")
	if def.Sketches().Count() != 1 {
		t.Fatalf("%d sketches, want 1", def.Sketches().Count())
	}
	sk := def.Sketches().Item(0)
	if sk.Lines().Count() != 4 || sk.Arcs().Count() != 4 {
		t.Errorf("rounded rect: %d lines %d arcs, want 4/4", sk.Lines().Count(), sk.Arcs().Count())
	}
}
