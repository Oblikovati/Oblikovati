// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
)

// TestSetLengthUnitCentresEmptyDocument is the ADR-0042 Phase 2 activation (#1259): choosing an
// extreme length unit on a fresh part centres the working scale on it (so coordinates stay O(1)),
// while a band unit (mm…ft) keeps the centimetre default — existing documents are unchanged.
func TestSetLengthUnitCentresEmptyDocument(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		unit string
		want float64
	}{{"µm", 1e-4}, {"km", 1e5}, {"mm", 1}, {"in", 1}} {
		d := NewPartComponentDefinition()
		if err := d.SetLengthUnit(tc.unit); err != nil {
			t.Fatalf("SetLengthUnit(%q): %v", tc.unit, err)
		}
		if got := d.WorkingScale(); got != tc.want {
			t.Errorf("after SetLengthUnit(%q) WorkingScale = %v, want %v", tc.unit, got, tc.want)
		}
	}
}

// TestSetLengthUnitLeavesModeledDocumentAlone is the guard: once geometry exists, changing the
// length unit must NOT re-scale the working scale (that would reinterpret stored coordinates).
func TestSetLengthUnitLeavesModeledDocumentAlone(t *testing.T) {
	t.Parallel()
	d := partWithBlock(t, math.P3(0, 0, 0), math.P3(2, 2, 2)) // centimetre block, has a feature
	if err := d.SetLengthUnit("µm"); err != nil {
		t.Fatal(err)
	}
	if got := d.WorkingScale(); got != 1 {
		t.Errorf("modeled document re-scaled to %v; want unchanged 1 (display-only unit change)", got)
	}
}

// TestSetUnitsCentresEmptyDocument covers the dialog path (SetUnits): an incoming default-scale
// units object on a fresh part centres on its length unit, while an explicitly-centred one
// (CenteredOnLength) is respected.
func TestSetUnitsCentresEmptyDocument(t *testing.T) {
	t.Parallel()
	// Dialog-style: default working scale, µm preferred ⇒ auto-centres.
	u := param.DefaultUnitsOfMeasure().Clone()
	if err := u.SetPreferred(param.Length, "µm"); err != nil {
		t.Fatal(err)
	}
	d := NewPartComponentDefinition()
	d.SetUnits(u)
	if got := d.WorkingScale(); got != 1e-4 {
		t.Errorf("SetUnits(µm, default scale) ⇒ WorkingScale %v, want 1e-4", got)
	}

	// Explicit working scale is respected, not overridden.
	ex, err := param.DefaultUnitsOfMeasure().CenteredOnLength("km")
	if err != nil {
		t.Fatal(err)
	}
	d2 := NewPartComponentDefinition()
	d2.SetUnits(ex)
	if got := d2.WorkingScale(); got != 1e5 {
		t.Errorf("SetUnits(explicit km scale) ⇒ WorkingScale %v, want 1e5", got)
	}
}

// TestActivationLetsPicometreDocumentBuild is the end-to-end proof of the activation: on a part
// centred on picometres, a bare "5" parses to an O(1) working coordinate, so a primitive builds
// as a valid solid — whereas the same physical 5 pm in a centimetre document is ~5e-10 and the
// primitive collapses (the underflow ADR-0042 Phase 1 could not reach). This is why Phase 2 makes
// the nm/pm semiconductor scales usable.
func TestActivationLetsPicometreDocumentBuild(t *testing.T) {
	t.Parallel()
	pm := NewPartComponentDefinition()
	if err := pm.SetLengthUnit("pm"); err != nil {
		t.Fatal(err)
	}
	q, err := pm.Units().Parse("5", param.Length) // bare ⇒ working (pm) unit
	if err != nil {
		t.Fatal(err)
	}
	if q.Value < 0.9 || q.Value > 5.1 { // O(1), not 5e-10
		t.Fatalf("pm document: bare 5 parsed to %v working units, want ~5 (O(1))", q.Value)
	}
	body, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(q.Value, q.Value, q.Value), "pmbox")
	if err != nil {
		t.Fatalf("pm box build: %v", err)
	}
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("pm box is not a valid solid: %+v", r)
	}

	// Control: the same physical size in a centimetre document underflows to a degenerate box.
	cm := NewPartComponentDefinition() // working scale 1 (cm)
	qcm, _ := cm.Units().Parse("5 pm", param.Length)
	if qcm.Value > 1e-6 {
		t.Fatalf("cm document: 5 pm parsed to %v, want ~5e-10 (the underflow case)", qcm.Value)
	}
}
