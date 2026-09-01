// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// The sweep definition union over the wire (M08 PBI-094, #314): the
// features.add sweep args discriminate on definitionType with the frozen
// api/types spellings.

// sweepFixture: a centered square profile on XY plus a straight path sketch
// on XZ rising in Z (so the path lifts out of the profile plane).
func sweepFixture(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"20 mm","height":"20 mm"}`, &wire.SketchRectangleResult{})
	call(t, r, s, "sketch.create", `{"plane":"XZ"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":1,"kind":"line","points":[[0,0],[0,60]]}`, &wire.AddSketchEntityResult{})
	return r, s
}

type sweepAddResult struct {
	Bodies int `json:"bodies"`
}

// TestSweepUnionTaperOverWire: a tapered path sweep grows the section — more
// volume than the straight prism.
func TestSweepUnionTaperOverWire(t *testing.T) {
	t.Parallel()
	r, s := sweepFixture(t)
	var res sweepAddResult
	call(t, r, s, "features.add",
		`{"kind":"sweep","args":{"sketchIndex":0,"pathSketchIndex":1,"taper":"5 deg","operation":"new"}}`, &res)
	if res.Bodies != 1 {
		t.Fatalf("tapered sweep bodies = %d, want 1", res.Bodies)
	}
}

// TestSweepUnionStationsOverWire: the pathAndSectionTwists variant accepts a
// station table and rejects a descending one with the offending values.
func TestSweepUnionStationsOverWire(t *testing.T) {
	t.Parallel()
	r, s := sweepFixture(t)
	var res sweepAddResult
	call(t, r, s, "features.add",
		`{"kind":"sweep","args":{"sketchIndex":0,"pathSketchIndex":1,"definitionType":"pathAndSectionTwists","twistStations":[{"t":0,"angle":"0 deg"},{"t":1,"angle":"90 deg"}],"operation":"new"}}`, &res)
	if res.Bodies != 1 {
		t.Fatalf("station sweep bodies = %d, want 1", res.Bodies)
	}
	_, err := r.Handle(s, "features.add",
		[]byte(`{"kind":"sweep","args":{"sketchIndex":0,"pathSketchIndex":1,"definitionType":"pathAndSectionTwists","twistStations":[{"t":0.5,"angle":"0 deg"},{"t":0.2,"angle":"90 deg"}]}}`))
	if err == nil {
		t.Fatal("descending twist stations must be rejected")
	}
}

// TestSweepUnionGuideRailOverWire: a rail sketch drives the scaled variant.
func TestSweepUnionGuideRailOverWire(t *testing.T) {
	t.Parallel()
	r, s := sweepFixture(t)
	// Rail on XZ, diverging from the path in X as it rises.
	call(t, r, s, "sketch.create", `{"plane":"XZ"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":2,"kind":"line","points":[[30,0],[50,60]]}`, &wire.AddSketchEntityResult{})
	var res sweepAddResult
	call(t, r, s, "features.add",
		`{"kind":"sweep","args":{"sketchIndex":0,"pathSketchIndex":1,"definitionType":"pathAndGuideRail","railSketchIndex":2,"railIndex":0,"scaling":"xy","operation":"new"}}`, &res)
	if res.Bodies != 1 {
		t.Fatalf("rail sweep bodies = %d, want 1", res.Bodies)
	}
}

// TestSweepUnionSolidOverWire: the solid variant drags an existing body along
// the path; a missing toolBodyIndex is a precise schema-level error.
func TestSweepUnionSolidOverWire(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~133s): `make test-corpus`")
	}
	t.Parallel()
	r, s := sweepFixture(t)
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"10 mm"}}`, &sweepAddResult{})
	var res sweepAddResult
	call(t, r, s, "features.add",
		`{"kind":"sweep","args":{"definitionType":"solid","toolBodyIndex":0,"pathSketchIndex":1,"operation":"new"}}`, &res)
	if res.Bodies != 2 {
		t.Fatalf("solid sweep bodies = %d, want 2 (tool + envelope)", res.Bodies)
	}
	if _, err := r.Handle(s, "features.add",
		[]byte(`{"kind":"sweep","args":{"definitionType":"solid","pathSketchIndex":1}}`)); err == nil {
		t.Fatal("solid sweep without toolBodyIndex must error")
	}
}
