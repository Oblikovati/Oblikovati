// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
)

// planeListed reports whether a plane ref is present in the list (construction datums included).
func planeListed(t *testing.T, r *Router, s *app.Session, ref string) bool {
	t.Helper()
	var planes wire.ListWorkPlanesResult
	call(t, r, s, "workPlanes.list", `{"includeConstruction":true}`, &planes)
	return findPlane(planes.Planes, ref).Ref == ref
}

// makeConstructionPlaneWithSketch builds a construction work plane and a sketch hosted on it,
// returning the plane result and the sketch index.
func makeConstructionPlaneWithSketch(t *testing.T, r *Router, s *app.Session, construction bool) (wire.CreateWorkPlaneResult, int) {
	t.Helper()
	var pl wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create",
		fmt.Sprintf(`{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"10 mm","construction":%v}`, construction), &pl)
	var sk wire.CreateSketchResult
	call(t, r, s, "sketch.create", fmt.Sprintf(`{"workPlaneIndex":%d}`, pl.Index), &sk)
	return pl, sk.SketchIndex
}

// TestConstructionPlaneAutoDeletesWithLastConsumer: a construction work plane hosting a single
// sketch is auto-deleted when that sketch (its last consumer) is deleted (#1849).
func TestConstructionPlaneAutoDeletesWithLastConsumer(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	pl, sketchIndex := makeConstructionPlaneWithSketch(t, r, s, true)
	if !planeListed(t, r, s, pl.Ref) {
		t.Fatal("construction plane should exist while its sketch consumes it")
	}
	var ok wire.OKResult
	call(t, r, s, "sketch.delete", fmt.Sprintf(`{"sketchIndex":%d}`, sketchIndex), &ok)
	if planeListed(t, r, s, pl.Ref) {
		t.Error("construction plane should auto-delete when its last consumer (the sketch) is deleted")
	}
}

// TestNonConstructionPlaneSurvivesConsumerDelete: a plain (non-construction) work plane is never
// auto-deleted — deleting its sketch leaves the datum in place (#1849).
func TestNonConstructionPlaneSurvivesConsumerDelete(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	pl, sketchIndex := makeConstructionPlaneWithSketch(t, r, s, false)
	var ok wire.OKResult
	call(t, r, s, "sketch.delete", fmt.Sprintf(`{"sketchIndex":%d}`, sketchIndex), &ok)
	if !planeListed(t, r, s, pl.Ref) {
		t.Error("a non-construction plane must never be auto-deleted")
	}
}

// TestConstructionPlaneRetainedByDatumConsumer: a construction plane consumed by another work datum
// (a work axis normal to it) is retained even when a sketch on it is deleted — the datum consumer
// still references it (#1849).
func TestConstructionPlaneRetainedByDatumConsumer(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	pl, sketchIndex := makeConstructionPlaneWithSketch(t, r, s, true)
	// A work point + axis normal to the construction plane keeps consuming it after the sketch goes.
	var pt wire.CreateWorkPointResult
	call(t, r, s, "workPoints.create", `{"at":[0,0,1]}`, &pt)
	var ax wire.CreateWorkAxisResult
	call(t, r, s, "workAxes.create", fmt.Sprintf(`{"kind":"point-and-plane","refs":["%s","%s"]}`, pt.Ref, pl.Ref), &ax)

	var ok wire.OKResult
	call(t, r, s, "sketch.delete", fmt.Sprintf(`{"sketchIndex":%d}`, sketchIndex), &ok)
	if !planeListed(t, r, s, pl.Ref) {
		t.Error("construction plane must be retained while a work axis still references it")
	}
}

// TestConstructionPlaneRetainedByFeatureConsumer: a construction plane a part feature (a solid
// split) references is retained when a *different* consumer (a sketch on it) is deleted — the recipe
// scan sees the feature still holds "plane/N" — and is auto-deleted only once the split too is gone.
// This is the critical safety property: a feature-consumed datum is never wrongly pruned (#1849).
func TestConstructionPlaneRetainedByFeatureConsumer(t *testing.T) {
	t.Parallel()
	r, s := boxPartSession(t) // a box (sketch0 + extrude); all features marshalable
	var pl wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create", `{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"20 mm","construction":true}`, &pl)
	var sk wire.CreateSketchResult
	call(t, r, s, "sketch.create", fmt.Sprintf(`{"workPlaneIndex":%d}`, pl.Index), &sk)
	call(t, r, s, "features.add", fmt.Sprintf(`{"kind":"splitSolid","args":{"workPlaneIndex":%d,"type":"splitBody"}}`, pl.Index), &struct{}{})
	part := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	splitID := uint64(part.Features().Item(part.Features().Count() - 1).ID())

	// Deleting the sketch must NOT prune the plane: the split still references it.
	var ok wire.OKResult
	call(t, r, s, "sketch.delete", fmt.Sprintf(`{"sketchIndex":%d}`, sk.SketchIndex), &ok)
	if !planeListed(t, r, s, pl.Ref) {
		t.Fatal("construction plane must be retained while the split feature references it (recipe scan)")
	}
	// Once the split is deleted too, the plane has no consumer and is auto-deleted.
	var del wire.DeleteFeatureResult
	call(t, r, s, "features.delete", fmt.Sprintf(`{"id":%d}`, splitID), &del)
	if planeListed(t, r, s, pl.Ref) {
		t.Error("construction plane should auto-delete once its last consumer (the split) is gone")
	}
}
