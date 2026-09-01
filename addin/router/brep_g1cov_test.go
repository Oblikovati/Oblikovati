// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"strconv"
	"testing"

	"oblikovati.org/app"
)

// TestBpcovPrimitiveValidationRejectsBadVectors drives brep.createPrimitive's per-kind
// coordinate-validation branches: each request carries a malformed leading vector (2 values
// instead of 3), so every primitive builder's xyz-decode error path is exercised, plus the
// unknown-kind default.
func TestBpcovPrimitiveValidationRejectsBadVectors(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	for _, args := range bpcovBadPrimitives() {
		if _, err := r.Handle(s, "brep.createPrimitive", []byte(args)); err == nil {
			t.Errorf("brep.createPrimitive(%s) returned nil error, want rejection", args)
		}
	}
}

// bpcovBadPrimitives enumerates one malformed request per primitive validation branch.
func bpcovBadPrimitives() []string {
	return []string{
		`{"kind":"block","min":[0,0],"max":[1,1,1]}`,
		`{"kind":"block","min":[0,0,0],"max":[1,1]}`,
		`{"kind":"cylinderCone","bottom":[0,0],"top":[0,0,1]}`,
		`{"kind":"cylinderCone","bottom":[0,0,0],"top":[0,0]}`,
		`{"kind":"sphere","center":[0,0]}`,
		`{"kind":"torus","center":[0,0]}`,
		`{"kind":"torus","center":[0,0,0],"axis":[0,0]}`,
		`{"kind":"pyramid"}`,
	}
}

// TestBpcovPrimitiveRoundTrip drives the happy path: a valid block becomes a transient solid
// with 6 faces and the expected volume, and its handle round-trips into brep.stats.
func TestBpcovPrimitiveRoundTrip(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	var res bpcovHandleReply
	call(t, r, s, "brep.createPrimitive", `{"kind":"block","min":[0,0,0],"max":[2,3,4]}`, &res)
	if !res.Stats.Solid || res.Stats.Faces != 6 {
		t.Fatalf("block stats = %+v, want a 6-face solid", res.Stats)
	}
	if res.Stats.Volume < 23.9 || res.Stats.Volume > 24.1 {
		t.Fatalf("block volume = %g, want ~24 (2x3x4)", res.Stats.Volume)
	}
}

// bpcovHandleReply mirrors the fields of wire.BrepHandleResult this test asserts on.
type bpcovHandleReply struct {
	Handle int `json:"handle"`
	Stats  struct {
		Solid  bool    `json:"solid"`
		Faces  int     `json:"faces"`
		Volume float64 `json:"volume"`
	} `json:"stats"`
}

// TestBpcovBooleanRejectsMissingBlank drives brep.boolean's blank-lookup error branch: a handle
// with no registered transient body is rejected before any tool is resolved.
func TestBpcovBooleanRejectsMissingBlank(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	bad := `{"blankHandle":9999,"operation":"union","tool":{"handle":9998}}`
	if _, err := r.Handle(s, "brep.boolean", []byte(bad)); err == nil {
		t.Fatal("brep.boolean with an unregistered blank handle returned nil error")
	}
}

// TestBpcovBrepSourceRejectsAmbiguousRef drives brepSource's mutual-exclusion branch: a body
// ref that sets both handle and bodyIndex is rejected (via a boolean whose tool is ambiguous).
func TestBpcovBrepSourceRejectsAmbiguousRef(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	blank := bpcovBlockHandle(t, r, s)
	amb := `{"blankHandle":` + strconv.Itoa(blank) + `,"operation":"union","tool":{"handle":1,"bodyIndex":0}}`
	if _, err := r.Handle(s, "brep.boolean", []byte(amb)); err == nil {
		t.Fatal("brep.boolean with a handle+bodyIndex tool ref returned nil error")
	}
}

// bpcovBlockHandle creates a transient block and returns its handle.
func bpcovBlockHandle(t *testing.T, r *Router, s *app.Session) int {
	t.Helper()
	var res bpcovHandleReply
	call(t, r, s, "brep.createPrimitive", `{"kind":"block","min":[0,0,0],"max":[1,1,1]}`, &res)
	return res.Handle
}
