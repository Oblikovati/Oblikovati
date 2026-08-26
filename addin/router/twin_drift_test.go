// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"reflect"
	"sort"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/math"
	"oblikovati.org/model/assembly"
	"oblikovati.org/model/sheetmetal"
)

// These guards protect the hand-converted model↔wire twin structs the 2026-07 audit flagged
// (B9, #1620): a field added to one side compiles fine while the hand converter silently drops
// it on the other, producing a partly-populated DTO no test notices. The alias pattern can't
// collapse these — the model side speaks kernel types (math.Matrix4) and the wire side the
// api's serialization types (types.Matrix, json tags) across the ADR-0018 dependency wall — so
// each twin is guarded here instead. Red-verify by adding a phantom field to either side of a
// pair: TestModelWireTwinsHaveMatchingFields then names it.

// twinPair is one model/wire twin whose exported field NAMES must stay in lockstep.
type twinPair struct {
	name        string
	model, wire reflect.Type
}

// modelWireTwins is the registry of hand-converted twins (add a row when a new one appears).
var modelWireTwins = []twinPair{
	{"DriveResult", reflect.TypeFor[assembly.DriveResult](), reflect.TypeFor[wire.DriveResult]()},
	{"DriveFrame", reflect.TypeFor[assembly.DriveFrame](), reflect.TypeFor[wire.DriveFrame]()},
	{"Placement", reflect.TypeFor[assembly.OccurrencePlacement](), reflect.TypeFor[wire.DrivePlacement]()},
	{"FlatPatternSettings", reflect.TypeFor[sheetmetal.FlatPatternSettings](), reflect.TypeFor[wire.FlatPatternSettings]()},
}

// TestModelWireTwinsHaveMatchingFields fails when the exported field-name sets of a twin pair
// diverge — the drift signal, caught before someone forgets to update the converter.
func TestModelWireTwinsHaveMatchingFields(t *testing.T) {
	for _, tw := range modelWireTwins {
		got, want := exportedFieldNames(tw.model), exportedFieldNames(tw.wire)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s twin field drift: model %v vs wire %v — update the twin and its hand "+
				"converter together (audit B9, #1620)", tw.name, got, want)
		}
	}
}

// exportedFieldNames returns t's exported field names in sorted order.
func exportedFieldNames(t reflect.Type) []string {
	var names []string
	for f := range t.Fields() {
		if f.IsExported() {
			names = append(names, f.Name)
		}
	}
	sort.Strings(names)
	return names
}

// TestDriveResultToWirePopulatesEveryField converts a fully non-zero model DriveResult and
// asserts every wire field carried over — so a converter that drops a newly added field fails
// here rather than shipping a half-empty DTO.
func TestDriveResultToWirePopulatesEveryField(t *testing.T) {
	cells := [16]math.Scalar{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	res := assembly.DriveResult{
		StoppedByCollision: true,
		StoppedAtStep:      3,
		Frames: []assembly.DriveFrame{{
			Value:      1.5,
			Collided:   true,
			Placements: []assembly.OccurrencePlacement{{Occurrence: 7, Transform: math.Matrix4FromCells(cells)}},
		}},
	}
	assertDriveResultCarried(t, res, driveResultToWire(res), cells)
}

// assertDriveResultCarried checks every field of the converted wire result matches the source.
func assertDriveResultCarried(t *testing.T, src assembly.DriveResult, got wire.DriveResult, cells [16]math.Scalar) {
	t.Helper()
	if !got.StoppedByCollision || got.StoppedAtStep != src.StoppedAtStep || len(got.Frames) != 1 {
		t.Fatalf("drive result header dropped: %+v", got)
	}
	f := got.Frames[0]
	if f.Value != 1.5 || !f.Collided || len(f.Placements) != 1 {
		t.Fatalf("drive frame fields dropped: %+v", f)
	}
	p := f.Placements[0]
	if p.Occurrence != 7 || p.Transform.Cells != cells {
		t.Errorf("placement dropped: occurrence=%d transform=%v", p.Occurrence, p.Transform.Cells)
	}
}

// TestFlatPatternSettingsRoundTrip proves the sheet-metal twin's 1:1 mapping is identity in
// both directions; paired with the field-parity guard, a dropped/renamed field is caught.
func TestFlatPatternSettingsRoundTrip(t *testing.T) {
	m := sheetmetal.FlatPatternSettings{DeferUpdate: true}
	w := wire.FlatPatternSettings{DeferUpdate: m.DeferUpdate}
	if back := (sheetmetal.FlatPatternSettings{DeferUpdate: w.DeferUpdate}); back != m {
		t.Errorf("flat-pattern settings round-trip = %+v, want %+v", back, m)
	}
}
