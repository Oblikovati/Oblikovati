// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"math"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

func TestWorkPointsCreateRejectsBadPosition(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "workPoints.create", []byte(`{"at":[1,2]}`)); err == nil {
		t.Error("a 2-component position must error")
	}
}

// TestWorkPointsPlaneAxisIntersection: the Z axis pierces the XY plane at the origin. The created
// point is healthy and lands at (0,0,0) — proving the plane-axis-intersection kind reaches the
// model's AddByPlaneAndAxisIntersection (#1842).
func TestWorkPointsPlaneAxisIntersection(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	var res wire.CreateWorkPointResult
	call(t, r, s, "workPoints.create",
		`{"kind":"plane-axis-intersection","refs":["origin/plane/xy","origin/axis/z"]}`, &res)
	if !res.Healthy {
		t.Fatalf("Z∩XY point not healthy: %+v", res)
	}
	at := lastWorkPoint(t, s).Point()
	if math.Abs(float64(at.X)) > 1e-9 || math.Abs(float64(at.Y)) > 1e-9 || math.Abs(float64(at.Z)) > 1e-9 {
		t.Errorf("Z∩XY point at %v, want the origin", at)
	}
}

// TestWorkPointsPlaneAxisParallelIsUnhealthy: an axis parallel to the plane (X axis, XY plane)
// has no intersection — the point is still created, but reported healthy=false with a reason,
// not a bare error.
func TestWorkPointsPlaneAxisParallelIsUnhealthy(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	var res wire.CreateWorkPointResult
	call(t, r, s, "workPoints.create",
		`{"kind":"plane-axis-intersection","refs":["origin/plane/xy","origin/axis/x"]}`, &res)
	if res.Healthy || res.Reason == "" {
		t.Errorf("X∥XY point should be unhealthy with a reason, got %+v", res)
	}
}

func TestWorkPointsPlaneAxisWrongReferenceCount(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "workPoints.create",
		[]byte(`{"kind":"plane-axis-intersection","refs":["origin/plane/xy"]}`)); err == nil {
		t.Error("plane-axis-intersection with 1 reference must error")
	}
}

func TestWorkPointsCreateUnknownKind(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "workPoints.create", []byte(`{"kind":"no-such-kind"}`)); err == nil {
		t.Error("expected error for unknown work-point kind")
	}
}

// lastWorkPoint returns the most recently created work point of the active part.
func lastWorkPoint(t *testing.T, s *app.Session) *feature.WorkPoint {
	t.Helper()
	pts := s.ActiveDocument().Content().(*compdef.PartComponentDefinition).WorkPoints()
	if pts.Count() == 0 {
		t.Fatal("no work points")
	}
	return pts.Item(pts.Count() - 1)
}
