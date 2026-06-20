// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	stdmath "math"
	"testing"

	"oblikovati.org/api/wire"
)

// faceEvalArgs marshals face-evaluate args to a JSON string (reference keys carry raw bytes,
// so they must be JSON-escaped via Marshal, not concatenated).
func faceEvalArgs(t *testing.T, args wire.FaceEvaluateArgs) string {
	t.Helper()
	b, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	return string(b)
}

// TestBodyFaceEvaluate covers the batched face-evaluation modes on the box's top face: a
// point projected onto it lands on the plane (z≈5 cm), and re-evaluating the returned (u,v)
// yields the plane's up normal. Exercises closestPoint → normalAtParam.
func TestBodyFaceEvaluate(t *testing.T) {
	r, s := boxPartSession(t)

	var keys wire.ReferenceKeysResult
	call(t, r, s, "model.referenceKeys", `{}`, &keys)
	if len(keys.Bodies) == 0 {
		t.Fatal("no bodies in reference keys")
	}
	topKey := ""
	topZ := stdmath.Inf(-1)
	for _, face := range keys.Bodies[0].Faces {
		if face.Kind == "plane" && len(face.Point) == 3 && face.Point[2] > topZ {
			topZ, topKey = face.Point[2], face.Key
		}
	}
	if topKey == "" {
		t.Fatal("no top plane face found")
	}

	// closestPoint: project a point 5 cm above the top-face centre — it must land on z≈5.
	var closest wire.FaceEvaluateResult
	call(t, r, s, "body.faceEvaluate", faceEvalArgs(t, wire.FaceEvaluateArgs{
		BodyIndex: 0, FaceKey: topKey, Mode: wire.FaceEvalClosestPoint, Inputs: []float64{2, 1.5, 10},
	}), &closest)
	if len(closest.Points) != 3 || stdmath.Abs(closest.Points[2]-5) > 1e-6 {
		t.Fatalf("closestPoint = %v, want a point on z=5", closest.Points)
	}
	if len(closest.UVs) != 2 {
		t.Fatalf("closestPoint UVs = %v, want one (u,v) pair", closest.UVs)
	}
	if len(closest.ParamRange) != 4 {
		t.Errorf("paramRange = %v, want [uMin,vMin,uMax,vMax]", closest.ParamRange)
	}

	// normalAtParam at the projected (u,v): the top face's outward normal is ±Z.
	var normal wire.FaceEvaluateResult
	call(t, r, s, "body.faceEvaluate", faceEvalArgs(t, wire.FaceEvaluateArgs{
		BodyIndex: 0, FaceKey: topKey, Mode: wire.FaceEvalNormalAtParam, Inputs: closest.UVs,
	}), &normal)
	if len(normal.Normals) != 3 || stdmath.Abs(stdmath.Abs(normal.Normals[2])-1) > 1e-6 {
		t.Fatalf("normalAtParam = %v, want ±Z", normal.Normals)
	}
	if stdmath.Abs(normal.Points[2]-5) > 1e-6 {
		t.Errorf("normalAtParam point z = %g, want 5", normal.Points[2])
	}
}

// TestBodyFaceEvaluateErrors covers the unknown-mode and unknown-face-key guards.
func TestBodyFaceEvaluateErrors(t *testing.T) {
	r, s := boxPartSession(t)
	badKey := faceEvalArgs(t, wire.FaceEvaluateArgs{BodyIndex: 0, FaceKey: "nope", Mode: wire.FaceEvalClosestPoint, Inputs: []float64{0, 0, 0}})
	if err := tryCall(t, r, s, "body.faceEvaluate", badKey); err == nil {
		t.Error("an unknown face key must error")
	}

	var keys wire.ReferenceKeysResult
	call(t, r, s, "model.referenceKeys", `{}`, &keys)
	badMode := faceEvalArgs(t, wire.FaceEvaluateArgs{BodyIndex: 0, FaceKey: keys.Bodies[0].Faces[0].Key, Mode: "bogus", Inputs: []float64{0, 0}})
	if err := tryCall(t, r, s, "body.faceEvaluate", badMode); err == nil {
		t.Error("an unknown mode must error")
	}
}
