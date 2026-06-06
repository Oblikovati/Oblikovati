// SPDX-License-Identifier: GPL-2.0-only

package exchange_test

import (
	"path/filepath"
	"testing"

	"oblikovati/api/types"
	"oblikovati/kernel/ops"
	"oblikovati/model/compdef"
	"oblikovati/model/exchange"
)

// stepFixture is a hand-authored AP203 fixture shared with the kernel step tests.
func stepFixture(name string) string {
	return filepath.Join("..", "..", "kernel", "exchange", "step", "testdata", name)
}

// TestImportStepRoutesToBrep checks the unified dispatch routes a .step file through the STEP
// reader (not the mesh reader) and lands a valid solid in the part.
func TestImportStepRoutesToBrep(t *testing.T) {
	part := compdef.NewPartComponentDefinition()
	res, err := exchange.Import(part, stepFixture("cube.step"), types.FormatSTEP)
	if err != nil {
		t.Fatalf("Import step: %v", err)
	}
	if res.BodyCount < 1 || !res.Solid {
		t.Fatalf("step import: bodyCount=%d solid=%v, want >=1 solid", res.BodyCount, res.Solid)
	}
	b := part.SurfaceBodies().Item(0)
	if r := ops.Validate(b); !r.Valid || !b.IsSolid() {
		t.Fatalf("imported step body not a valid solid: %+v", r)
	}
}

// TestStepRoundTripThroughDispatch imports a STEP solid, exports the part back to STEP, and
// re-imports — the body stays a valid solid with the same volume (the dispatch wires both ways).
func TestStepRoundTripThroughDispatch(t *testing.T) {
	src := compdef.NewPartComponentDefinition()
	if _, err := exchange.Import(src, stepFixture("cube.step"), types.FormatSTEP); err != nil {
		t.Fatalf("import: %v", err)
	}
	v0 := ops.BodyGeometryProperties(src.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume

	out := filepath.Join(t.TempDir(), "roundtrip.step")
	if _, err := exchange.Export(src, out, types.FormatSTEP, ""); err != nil {
		t.Fatalf("export: %v", err)
	}
	back := compdef.NewPartComponentDefinition()
	if _, err := exchange.Import(back, out, types.FormatSTEP); err != nil {
		t.Fatalf("re-import: %v", err)
	}
	b := back.SurfaceBodies().Item(0)
	if r := ops.Validate(b); !r.Valid || !b.IsSolid() {
		t.Fatalf("round-tripped step body not a valid solid: %+v", r)
	}
	if v1 := ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume; relErr(v0, v1) > 0.02 {
		t.Errorf("step round-trip volume changed: %.4f → %.4f", v0, v1)
	}
}

// TestFormatFromPath maps extensions to formats (case-insensitive), including STEP's two spellings.
func TestFormatFromPath(t *testing.T) {
	cases := map[string]types.ExchangeFormat{
		"a.stl": types.FormatSTL, "b.OBJ": types.FormatOBJ, "c.3mf": types.Format3MF,
		"d.step": types.FormatSTEP, "e.STP": types.FormatSTEP,
	}
	for path, want := range cases {
		if got, ok := exchange.FormatFromPath(path); !ok || got != want {
			t.Errorf("FormatFromPath(%q) = %q,%v; want %q", path, got, ok, want)
		}
	}
	if _, ok := exchange.FormatFromPath("x.dwg"); ok {
		t.Error("FormatFromPath accepted an unknown extension")
	}
}

func relErr(a, b float64) float64 {
	if a == 0 {
		return b
	}
	d := (a - b) / a
	if d < 0 {
		return -d
	}
	return d
}
