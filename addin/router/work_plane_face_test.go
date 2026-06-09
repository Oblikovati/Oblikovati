// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"testing"

	"oblikovati.org/api/wire"
)

// TestWorkPlaneOnFaceThenSketch covers the feature-interaction path: a work plane built on a
// body face a feature created (toWorkRefs tags the topology key as a FaceRef), then a sketch
// on that work plane (sketch.create workPlaneIndex). Both were impossible before — the work
// plane could only offset an origin plane, and a sketch could only sit on an origin plane.
func TestWorkPlaneOnFaceThenSketch(t *testing.T) {
	r, s := seededSession(t) // a part with one closed rectangle profile

	// Extrude the profile into a body so it has B-rep faces.
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"10 mm"}}`, nil)

	var keys wire.ReferenceKeysResult
	call(t, r, s, "model.referenceKeys", `{}`, &keys)
	if len(keys.Bodies) == 0 || len(keys.Bodies[0].Faces) == 0 {
		t.Fatal("extruded body exposes no faces")
	}
	faceKey := keys.Bodies[0].Faces[0].Key

	// Work plane offset from that feature-created face (FaceRef resolution).
	var wp wire.CreateWorkPlaneResult
	args := fmt.Sprintf(`{"kind":"plane-offset","refs":[%s],"offset":"5 mm"}`, jsonString(faceKey))
	call(t, r, s, "workPlanes.create", args, &wp)
	if !wp.Healthy {
		t.Fatalf("work plane on a feature face is unhealthy (index %d)", wp.Index)
	}

	// Sketch on that work plane.
	var sk wire.CreateSketchResult
	call(t, r, s, "sketch.create", fmt.Sprintf(`{"workPlaneIndex":%d}`, wp.Index), &sk)
	if sk.SketchIndex < 1 {
		t.Fatalf("sketch on work plane got index %d, want a new sketch beyond the seeded one", sk.SketchIndex)
	}
}

// jsonString quotes s as a JSON string literal (the reference key may contain control bytes).
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
