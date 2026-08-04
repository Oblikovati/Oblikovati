// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"testing"

	"oblikovati.org/api/types"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// A styled sketch must survive a DXF export/import cycle looking the same. Before #2015 the
// exporter wrote no formatting and the importer read none, so a round trip came back monochrome
// and continuous — which is what the issue means by DWG interoperability.
func TestSketchFormatSurvivesDXFRoundTrip(t *testing.T) {
	src := sketch.NewSketches().Add(sketch.XYPlane())
	red := src.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(10, 0))
	src.SetEntityFormat(red.EntityID(), sketch.EntityFormat{
		LineType: string(types.SketchLineDashed), Color: types.NewColor(255, 0, 0), LineWeight: 0.35,
	})
	plain := src.Lines().AddByTwoPoints(gmath.P2(0, 5), gmath.P2(10, 5))
	_ = plain

	data, n, err := ExportDXF(src, types.DXFR2018, param.DefaultUnitsOfMeasure())
	if err != nil {
		t.Fatalf("ExportDXF: %v", err)
	}
	if n != 2 {
		t.Fatalf("exported %d entities, want 2", n)
	}

	part := compdef.NewPartComponentDefinition()
	if _, err := importSketchFormat(part, types.FormatDXF, data, sketch.XYPlane()); err != nil {
		t.Fatalf("import: %v", err)
	}
	if part.Sketches().Count() != 1 {
		t.Fatalf("imported sketches = %d, want 1", part.Sketches().Count())
	}
	out := part.Sketches().Item(0)
	if got := out.EntityFormatCount(); got != 1 {
		t.Fatalf("restored format entries = %d, want exactly the one styled line", got)
	}
	var f sketch.EntityFormat
	for i := 0; i < out.Lines().Count(); i++ {
		if got, ok := out.EntityFormat(out.Lines().Item(i).EntityID()); ok {
			f = got
		}
	}
	if f.LineType != string(types.SketchLineDashed) {
		t.Errorf("line type = %q, want dashed", f.LineType)
	}
	if !f.Color.IsOverride() || f.Color.R != 255 || f.Color.G != 0 || f.Color.B != 0 {
		t.Errorf("colour = %+v, want opaque red", f.Color)
	}
	if f.LineWeight != 0.35 {
		t.Errorf("weight = %v, want 0.35 mm", f.LineWeight)
	}
}

// An unstyled sketch must export and import with no formatting at all, so the common case adds
// nothing to the file and nothing to the sketch.
func TestUnstyledSketchRoundTripsClean(t *testing.T) {
	src := sketch.NewSketches().Add(sketch.XYPlane())
	src.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(10, 0))

	data, _, err := ExportDXF(src, types.DXFR2018, param.DefaultUnitsOfMeasure())
	if err != nil {
		t.Fatalf("ExportDXF: %v", err)
	}
	part := compdef.NewPartComponentDefinition()
	if _, err := importSketchFormat(part, types.FormatDXF, data, sketch.XYPlane()); err != nil {
		t.Fatalf("import: %v", err)
	}
	if got := part.Sketches().Item(0).EntityFormatCount(); got != 0 {
		t.Errorf("format entries = %d, want 0", got)
	}
}
