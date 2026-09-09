// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The recorder's own rows (#3520). The corpus rows live in kernel/ops/query, where a real body proves
// the record reaches query.BodyMeshDiagnostics; these pin the four decisions the recorder itself makes.

// TestAChartDeclineIsRecordedOnWhateverTheFaceShipsInstead: the mesher that gave up returns no mesh, so
// the report can only ride the mesh the face fell back to — which is why the recorder sits at the router
// and not at the point of decline.
func TestAChartDeclineIsRecordedOnWhateverTheFaceShipsInstead(t *testing.T) {
	t.Parallel()
	log := &chartDeclineLog{}
	log.declined("the covering kept no triangle inside the chart's own window")
	m := log.recordOn(&Mesh{}, mustDeclineSurface(t))
	if !hasCode(m, CodeChartMesherDeclined) {
		t.Fatalf("a pending decline recorded nothing; the mesh carries %v", m.Diagnostics)
	}
	if !strings.Contains(m.Diagnostics[0].Detail, "kept no triangle") {
		t.Errorf("the record does not carry the reason: %q", m.Diagnostics[0].Detail)
	}
	if m.Diagnostics[0].Severity != diag.Defect {
		t.Errorf("a chart decline recorded at severity %v, want a Defect — it changes the shipped mesh",
			m.Diagnostics[0].Severity)
	}
}

// TestNothingIsRecordedForAFaceTheChartMesherNeverOwned guards the false-positive side. chartFaceMesh
// answers false for every aperiodic or unrecorded face in the model, which is the ORDINARY route onto
// the generic (u,v) trim path; a channel that fired there would fire on almost every curved face there
// is, and #2058's third acceptance is that a channel which cries on healthy geometry is ignored.
func TestNothingIsRecordedForAFaceTheChartMesherNeverOwned(t *testing.T) {
	t.Parallel()
	log := &chartDeclineLog{}
	if m := log.recordOn(&Mesh{}, mustDeclineSurface(t)); hasCode(m, CodeChartMesherDeclined) {
		t.Error("a face the mesher never owned was reported as a degradation")
	}
}

// TestTheFirstDeclineReasonWins: a face can reach chartFaceMesh twice — the classification's kindChart
// arm, then chartedTrimMesh once ToUVLoops fails — and the second call re-derives the same answer. The
// report names the reason the mesher first gave, not the last route that asked.
func TestTheFirstDeclineReasonWins(t *testing.T) {
	t.Parallel()
	log := &chartDeclineLog{}
	log.declined("the first reason")
	log.declined("the second reason")
	m := log.recordOn(&Mesh{}, mustDeclineSurface(t))
	if !strings.Contains(m.Diagnostics[0].Detail, "the first reason") {
		t.Errorf("the record reads %q, want the reason the mesher first gave", m.Diagnostics[0].Detail)
	}
}

// TestRecordingOnANilMeshIsSafe: a mesher can return nil, and the router stamps whatever it got. A nil
// mesh carries no diagnostics anywhere in this package, so the recorder must pass it through rather than
// panic on the recompute it is reporting on.
func TestRecordingOnANilMeshIsSafe(t *testing.T) {
	t.Parallel()
	log := &chartDeclineLog{}
	log.declined("the covering kept no triangle inside the chart's own window")
	if m := log.recordOn(nil, mustDeclineSurface(t)); m != nil {
		t.Errorf("recordOn(nil) returned %v, want nil", m)
	}
}

// mustDeclineSurface is the surface the recorder names in its report — any periodic surface will do,
// since the recorder reads only its type.
func mustDeclineSurface(t *testing.T) geom.Surface {
	t.Helper()
	s, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), wallR)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	return s
}
