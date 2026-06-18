// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"math"
	"testing"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// TestAnalysisMassPropertiesOverWire drives the mass-properties surface: the box-part fixture
// (a 40×30×50 mm block) reports its volume/area/mass through the live router→model→kernel stack.
func TestAnalysisMassPropertiesOverWire(t *testing.T) {
	r, s := boxPartSession(t)

	var mp wire.MassPropertiesResult
	call(t, r, s, "analysis.massProperties", `{"densityGCm3":2}`, &mp)

	// 40×30×50 mm = 60000 mm³; surface area 2(40·30+40·50+30·50) = 9400 mm²; mass = 2 g/cm³ × 60 cm³ = 120 g.
	if math.Abs(mp.VolumeMm3-60000) > 1 {
		t.Errorf("volume = %g mm³, want 60000", mp.VolumeMm3)
	}
	if math.Abs(mp.SurfaceAreaMm2-9400) > 1 {
		t.Errorf("surface area = %g mm², want 9400", mp.SurfaceAreaMm2)
	}
	if math.Abs(mp.MassG-120) > 1e-3 {
		t.Errorf("mass = %g g, want 120", mp.MassG)
	}
	// Inertia is populated; principal moments positive and sorted ascending.
	if mp.InertiaXxGmm2 <= 0 || mp.PrincipalMomentsGmm2[0] <= 0 || mp.PrincipalMomentsGmm2[0] > mp.PrincipalMomentsGmm2[2] {
		t.Errorf("inertia = %+v, want positive Ixx + ascending principal moments", mp)
	}

	// A bad accuracy errors; with no active part, the method errors.
	if _, err := r.Handle(s, "analysis.massProperties", []byte(`{"accuracy":"bogus"}`)); err == nil {
		t.Error("massProperties with a bad accuracy = ok, want error")
	}
	br, bs := New(opregistry.Default()), app.NewSession()
	if _, err := br.Handle(bs, "analysis.massProperties", []byte(`{}`)); err == nil {
		t.Error("massProperties with no active part = ok, want error")
	}
}

// TestAnalysisMeasureOverWire drives the measurement surface: an edge of the box-part fixture
// reports a length in {40, 30, 50} mm, and a face an area in {1200, 1500, 2000} mm².
func TestAnalysisMeasureOverWire(t *testing.T) {
	r, s := boxPartSession(t)
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatalf("ActivePart: %v", err)
	}
	body := part.SurfaceBodies().Item(0)
	edgeKey := string(body.Edges()[0].ReferenceKey())
	faceKey := string(body.Faces()[0].ReferenceKey())

	// Reference keys carry raw bytes (e.g. a leading \x02); marshal via a map so they are JSON-escaped,
	// exactly as the bridge sends them — string concatenation would emit invalid JSON.
	var m wire.MeasureResult
	call(t, r, s, "analysis.measure", measureArgs(t, "length", edgeKey, ""), &m)
	if m.Unit != "mm" || !nearAny(m.Value, 40, 30, 50) {
		t.Errorf("edge length = %+v, want one of 40/30/50 mm", m)
	}
	call(t, r, s, "analysis.measure", measureArgs(t, "area", faceKey, ""), &m)
	if m.Unit != "mm²" || !nearAny(m.Value, 1200, 1500, 2000) {
		t.Errorf("face area = %+v, want one of 1200/1500/2000 mm²", m)
	}
	vA := string(body.Vertices()[0].ReferenceKey())
	vB := string(body.Vertices()[1].ReferenceKey())
	call(t, r, s, "analysis.measure", measureArgs(t, "distance", vA, vB), &m)
	if m.Unit != "mm" || m.Value <= 0 {
		t.Errorf("vertex distance = %+v, want a positive mm distance", m)
	}
	if _, err := r.Handle(s, "analysis.measure", []byte(measureArgs(t, "distance", vA, ""))); err == nil {
		t.Error("distance with one vertex key = ok, want error")
	}

	// minDistance between two faces of the box equals one of its dimensions (opposite) or 0 (adjacent).
	allFaces := body.Faces()
	fA := string(allFaces[0].ReferenceKey())
	for i := 1; i < len(allFaces); i++ {
		fB := string(allFaces[i].ReferenceKey())
		call(t, r, s, "analysis.measure", measureArgs(t, "minDistance", fA, fB), &m)
		if m.Unit != "mm" || !nearAny(m.Value, 0, 30, 40, 50) {
			t.Errorf("minDistance(face0,face%d) = %+v, want one of 0/30/40/50 mm", i, m)
		}
	}
	if _, err := r.Handle(s, "analysis.measure", []byte(measureArgs(t, "minDistance", fA, "nope"))); err == nil {
		t.Error("minDistance with an unknown entity key = ok, want error")
	}
	if _, err := r.Handle(s, "analysis.measure", []byte(`{"type":"bogus","keyA":"00"}`)); err == nil {
		t.Error("measure with a bad type = ok, want error")
	}
	if _, err := r.Handle(s, "analysis.measure", []byte(`{"type":"length","keyA":"dead"}`)); err == nil {
		t.Error("measure with an unknown edge key = ok, want error")
	}

	// angle between two faces of the box is 90° (adjacent) or 180° (opposite).
	for i := 1; i < len(allFaces); i++ {
		call(t, r, s, "analysis.measure", measureArgs(t, "angle", fA, string(allFaces[i].ReferenceKey())), &m)
		if m.Unit != "deg" || (!nearAny(m.Value, 90) && !nearAny(m.Value, 180)) {
			t.Errorf("angle(face0,face%d) = %+v, want 90 or 180 deg", i, m)
		}
	}
	// angle on two vertices (no direction) errors.
	if _, err := r.Handle(s, "analysis.measure", []byte(measureArgs(t, "angle", vA, vB))); err == nil {
		t.Error("angle on two vertices = ok, want error")
	}
	// three-vertex angle is reported as a degree in [0,180].
	vC := string(body.Vertices()[2].ReferenceKey())
	call(t, r, s, "analysis.measure", threeKeyArgs(t, "angle", vA, vB, vC), &m)
	if m.Unit != "deg" || m.Value < 0 || m.Value > 180 {
		t.Errorf("three-point angle = %+v, want a degree in [0,180]", m)
	}

	// loopLength of a box face is one of the rectangular perimeters 2(w+h) ∈ {140,160,180} mm.
	call(t, r, s, "analysis.measure", measureArgs(t, "loopLength", fA, ""), &m)
	if m.Unit != "mm" || !nearAny(m.Value, 140, 160, 180) {
		t.Errorf("loopLength(face0) = %+v, want one of 140/160/180 mm", m)
	}
	if _, err := r.Handle(s, "analysis.measure", []byte(measureArgs(t, "loopLength", vA, ""))); err == nil {
		t.Error("loopLength on a vertex key = ok, want error")
	}
}

// threeKeyArgs JSON-encodes analysis.measure arguments with all three keys (raw-byte safe).
func threeKeyArgs(t *testing.T, measureType, keyA, keyB, keyC string) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{"type": measureType, "keyA": keyA, "keyB": keyB, "keyC": keyC})
	if err != nil {
		t.Fatalf("marshal measure args: %v", err)
	}
	return string(b)
}

// measureArgs JSON-encodes analysis.measure arguments through a map, so raw reference-key bytes are
// escaped exactly as the bridge sends them (an omitted keyB stays absent).
func measureArgs(t *testing.T, measureType, keyA, keyB string) string {
	t.Helper()
	args := map[string]any{"type": measureType, "keyA": keyA}
	if keyB != "" {
		args["keyB"] = keyB
	}
	b, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal measure args: %v", err)
	}
	return string(b)
}

// TestAnalysisModelHealthOverWire drives the model-health aggregation: a clean box reports ok with
// nothing to repair, and a suppressed feature is enumerated as suppressed.
func TestAnalysisModelHealthOverWire(t *testing.T) {
	r, s := boxPartSession(t)

	var clean wire.ModelHealthResult
	call(t, r, s, "analysis.modelHealth", `{}`, &clean)
	if clean.Overall != "ok" || clean.SickCount != 0 || len(clean.Unhealthy) != 0 {
		t.Errorf("clean box health = %+v, want ok with nothing unhealthy", clean)
	}

	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatalf("ActivePart: %v", err)
	}
	part.Features().Item(0).SetSuppressed(true)

	var suppressed wire.ModelHealthResult
	call(t, r, s, "analysis.modelHealth", `{}`, &suppressed)
	if len(suppressed.Unhealthy) != 1 || suppressed.Unhealthy[0].Status != "suppressed" {
		t.Errorf("after suppress = %+v, want one suppressed feature", suppressed)
	}
	if suppressed.Overall != "ok" {
		t.Errorf("overall after suppress = %q, want ok (suppression is neutral)", suppressed.Overall)
	}

	br, bs := New(opregistry.Default()), app.NewSession()
	if _, err := br.Handle(bs, "analysis.modelHealth", []byte(`{}`)); err == nil {
		t.Error("modelHealth with no active part = ok, want error")
	}
}

// nearAny reports whether v is within 0.01 of any of the candidate values.
func nearAny(v float64, candidates ...float64) bool {
	for _, c := range candidates {
		if math.Abs(v-c) < 0.01 {
			return true
		}
	}
	return false
}
