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
