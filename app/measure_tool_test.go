// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/analysis"
)

// sideFaceOf returns a vertical (horizontal-normal) planar face of the block.
func sideFaceOf(t *testing.T, b *topo.Body) *topo.Face {
	t.Helper()
	for _, f := range b.Faces() {
		if n := f.Geometry().NormalAt(0, 0); n.Z > -0.1 && n.Z < 0.1 {
			return f
		}
	}
	t.Fatal("no side face found")
	return nil
}

func faceEntity(f *topo.Face) measurePick   { return measurePick{analysis.MeasureEntity{Face: f}} }
func edgeEntity(e *topo.Edge) measurePick   { return measurePick{analysis.MeasureEntity{Edge: e}} }
func vertEntity(v *topo.Vertex) measurePick { return measurePick{analysis.MeasureEntity{Vertex: v}} }

// TestMeasureReadout checks the readout for each selection on a 40×40×20 mm block.
func TestMeasureReadout(t *testing.T) {
	_, block := newPartWithBlock(t, 4) // 4×4×2 cm = 40×40×20 mm
	top := topFaceOf(t, block)
	side := sideFaceOf(t, block)

	// A single face reports area and perimeter (top: 40×40 ⇒ 1600 mm², 160 mm).
	if r := measureReadout([]measurePick{faceEntity(top)}); !strings.Contains(r, "1600.000 mm²") || !strings.Contains(r, "160.000 mm") {
		t.Errorf("face readout = %q, want area 1600 mm² and perimeter 160 mm", r)
	}
	// A single edge reports a length (a box edge is 40 or 20 mm).
	if r := measureReadout([]measurePick{edgeEntity(block.Edges()[0])}); !strings.Contains(r, "edge length") {
		t.Errorf("edge readout = %q, want an edge length", r)
	}
	// Two adjacent faces (top + side) touch (min distance 0) at a right angle.
	if r := measureReadout([]measurePick{faceEntity(top), faceEntity(side)}); !strings.Contains(r, "min distance 0.000 mm") || !strings.Contains(r, "angle 90.00°") {
		t.Errorf("two-face readout = %q, want min distance 0 and angle 90°", r)
	}
	// Two vertices report a straight-line distance.
	vs := block.Vertices()
	if r := measureReadout([]measurePick{vertEntity(vs[0]), vertEntity(vs[1])}); !strings.HasPrefix(r, "distance ") {
		t.Errorf("two-vertex readout = %q, want a distance", r)
	}
	// Three vertices report the apex angle.
	if r := measureReadout([]measurePick{vertEntity(vs[0]), vertEntity(vs[1]), vertEntity(vs[2])}); !strings.Contains(r, "at the middle vertex") {
		t.Errorf("three-vertex readout = %q, want a three-point angle", r)
	}
}

// TestMeasureToolPick drives the tool through the picker: clicking a face yields its area readout
// and arms commit; a second non-vertex pick after capacity starts a fresh selection.
func TestMeasureToolPick(t *testing.T) {
	s, block := newPartWithBlock(t, 4)
	top := topFaceOf(t, block)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: top, Body: block}})

	m := NewMeasureTool()
	s.StartTool(m)
	s.Click(50, 50)
	if !m.CanCommit() {
		t.Fatal("measure tool not committable after one pick")
	}
	if !strings.Contains(m.Readout(), "face area") {
		t.Errorf("readout after face pick = %q, want a face area", m.Readout())
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
}

// TestMeasureToolMisc covers the tool's name/prompt/cancel and the readout edge cases.
func TestMeasureToolMisc(t *testing.T) {
	s, block := newPartWithBlock(t, 4)
	top, side := topFaceOf(t, block), sideFaceOf(t, block)
	v := block.Vertices()[0]

	m := NewMeasureTool()
	if m.Name() != "Measure" {
		t.Errorf("Name = %q, want Measure", m.Name())
	}
	if p := m.Prompt(nil); !strings.Contains(p, "pick a face") {
		t.Errorf("empty prompt = %q, want a pick instruction", p)
	}
	m.picks = []measurePick{vertEntity(v)}
	m.readout = "edge length 1.000 mm"
	if p := m.Prompt(nil); !strings.Contains(p, "pick more") {
		t.Errorf("active prompt = %q, want a pick-more hint", p)
	}

	// Readout edge cases: no picks, a lone vertex, and a three-pick fallback (not all vertices).
	if r := measureReadout(nil); !strings.Contains(r, "pick a face") {
		t.Errorf("empty readout = %q", r)
	}
	if r := measureReadout([]measurePick{vertEntity(v)}); !strings.Contains(r, "vertex") {
		t.Errorf("lone-vertex readout = %q, want a hint to pick more", r)
	}
	if r := measureReadout([]measurePick{faceEntity(top), faceEntity(side), vertEntity(v)}); !strings.Contains(r, "min distance") {
		t.Errorf("mixed three-pick readout = %q, want the pair fallback", r)
	}

	// A fourth pick is never accepted.
	m.picks = []measurePick{vertEntity(v), vertEntity(v), vertEntity(v)}
	if m.accepts(vertEntity(v)) {
		t.Error("a fourth pick should not be accepted")
	}
	// Cancel clears the picks and readout.
	m.Cancel(s)
	if len(m.picks) != 0 || m.readout != "" {
		t.Errorf("after Cancel: picks=%d readout=%q, want empty", len(m.picks), m.readout)
	}

	// Direct picks exercise every handle kind, the reset path, and an ignored kind.
	mp := NewMeasureTool()
	mp.Start(s)
	mp.Pick(s, EdgeHandle{Edge: block.Edges()[0]})
	mp.Pick(s, FaceHandle{Face: top, Body: block})  // 2 picks: edge + face
	mp.Pick(s, FaceHandle{Face: side, Body: block}) // 3rd non-vertex ⇒ restart at the new face
	if len(mp.picks) != 1 {
		t.Errorf("after a third non-vertex pick: %d picks, want 1 (restarted)", len(mp.picks))
	}
	mp.Pick(s, VertexHandle{Vertex: v})
	mp.Pick(s, BodyHandle{Body: block}) // unhandled kind ⇒ ignored
	if len(mp.picks) != 2 {
		t.Errorf("after a vertex then an ignored body pick: %d picks, want 2", len(mp.picks))
	}
}

// TestStartMeasureCommand checks the Inspect.Measure command activates the tool on a part and
// errors without one, and that ActiveMeasure reflects the running tool.
func TestStartMeasureCommand(t *testing.T) {
	s, _ := newPartWithBlock(t, 2)
	if err := startMeasure(s); err != nil {
		t.Fatalf("startMeasure: %v", err)
	}
	if s.ActiveMeasure() == nil {
		t.Error("Measure tool not active after startMeasure")
	}

	empty := NewSession()
	if err := startMeasure(empty); err == nil {
		t.Error("startMeasure with no active part = ok, want error")
	}
	if empty.ActiveMeasure() != nil {
		t.Error("ActiveMeasure with no tool = non-nil, want nil")
	}
}

// TestMeasureToolAcceptsResets checks that a third non-vertex pick restarts the selection.
func TestMeasureToolAcceptsResets(t *testing.T) {
	_, block := newPartWithBlock(t, 4)
	top, side := topFaceOf(t, block), sideFaceOf(t, block)
	m := &MeasureTool{picks: []measurePick{faceEntity(top), faceEntity(side)}}
	if m.accepts(faceEntity(top)) {
		t.Error("a third face pick should restart the selection, not extend it")
	}
	vs := block.Vertices()
	m2 := &MeasureTool{picks: []measurePick{vertEntity(vs[0]), vertEntity(vs[1])}}
	if !m2.accepts(vertEntity(vs[2])) {
		t.Error("a third vertex pick should extend to a three-point angle")
	}
}
