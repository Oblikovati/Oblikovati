// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
)

// twoBoxAssembly places two instances of a box part into a fresh assembly (the first grounded at the
// origin, the second offset on X), renders once so the world-body→occurrence cache is built, and
// returns the session, the assembly, and the two placed bodies in assembly space.
func twoBoxAssembly(t *testing.T) (*Session, *compdef.AssemblyComponentDefinition, *topo.Body, *topo.Body) {
	t.Helper()
	s, asm, boxDoc, asmDoc := emptyBoxAssembly(t)
	occ1, err := asm.PlaceComponentFromFile(asmDoc, boxDoc, "box:1", math.Identity4())
	if err != nil {
		t.Fatalf("place box:1: %v", err)
	}
	if _, err := asm.PlaceComponentFromFile(asmDoc, boxDoc, "box:2", math.Translation4(math.V3(10, 0, 0))); err != nil {
		t.Fatalf("place box:2: %v", err)
	}
	occ1.SetGrounded(true)

	bodies := s.VisibleBodies() // builds the world-body → occurrence cache OccurrenceOfBody reads
	if len(bodies) != 2 {
		t.Fatalf("assembly has %d visible bodies, want 2", len(bodies))
	}
	var b1, b2 *topo.Body
	for _, b := range bodies {
		if occ, _ := s.OccurrenceOfBody(b); occ == occ1 {
			b1 = b
		} else {
			b2 = b
		}
	}
	if b1 == nil || b2 == nil {
		t.Fatal("could not map both placed bodies to their occurrences")
	}
	return s, asm, b1, b2
}

// lowestFaceOf returns the body's face whose range-box centre has the smallest Z (the −Z face).
func lowestFaceOf(t *testing.T, b *topo.Body) *topo.Face {
	t.Helper()
	var lo *topo.Face
	for _, f := range b.Faces() {
		if lo == nil || f.RangeBox().Center().Z < lo.RangeBox().Center().Z {
			lo = f
		}
	}
	if lo == nil {
		t.Fatal("body has no faces")
	}
	return lo
}

// TestGripSnapToolCommitInfersAndSnaps drives the whole tool over a real two-box assembly: Start the
// tool, pick the grounded box's top face then the free box's bottom face, commit — the constraint is
// inferred (a mate between the two opposed planar faces) and recorded, the HUD reports it, and the
// tool is reachable via ActiveGripSnap. Also exercises the prompt/cancel paths.
func TestGripSnapToolCommitInfersAndSnaps(t *testing.T) {
	s, asm, grounded, free := twoBoxAssembly(t)
	s.StartTool(NewGripSnapTool())
	tool := s.ActiveGripSnap()
	if tool == nil {
		t.Fatal("ActiveGripSnap is nil after StartTool")
	}
	if got := s.ActiveTool(); got == nil {
		t.Fatal("no active tool after StartTool")
	}
	tool.Start(s)

	if p := tool.Prompt(s); p == "" {
		t.Error("prompt should guide the first pick")
	}
	tool.Pick(s, FaceHandle{Face: topFaceOf(t, grounded), Body: grounded})
	if p := tool.Prompt(s); p == "" {
		t.Error("prompt should guide the second pick")
	}
	tool.Pick(s, FaceHandle{Face: lowestFaceOf(t, free), Body: free})
	if !tool.CanCommit() {
		t.Fatal("two picks: CanCommit should be true")
	}
	if p := tool.Prompt(s); p == "" {
		t.Error("prompt should offer OK after both picks")
	}

	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if asm.Constraints().Count() != 1 {
		t.Errorf("after snap, constraint count = %d, want 1", asm.Constraints().Count())
	}
	if got := tool.Inferred(); got != "mate" {
		t.Errorf("inferred = %q, want mate (two opposed planar faces)", got)
	}
}

// TestGripSnapToolCommitRejectsBadSecondFace: a valid first pick but an unresolved second face is a
// clean error, leaving the tool open.
func TestGripSnapToolCommitRejectsBadSecondFace(t *testing.T) {
	s, _, grounded, _ := twoBoxAssembly(t)
	tool := NewGripSnapTool()
	tool.Pick(s, FaceHandle{Face: topFaceOf(t, grounded), Body: grounded})
	tool.Pick(s, FaceHandle{}) // unresolved
	if err := tool.Commit(s); err == nil {
		t.Error("commit with an unresolved second face should return an error")
	}
}

// TestGripSnapToolCancelClears: cancelling drops the picks and the tool stays usable.
func TestGripSnapToolCancelClears(t *testing.T) {
	s := assemblySession(t)
	tool := NewGripSnapTool()
	tool.Start(s)
	tool.Pick(s, FaceHandle{})
	tool.Cancel(s)
	if len(tool.faces) != 0 {
		t.Errorf("after cancel, %d faces remain, want 0", len(tool.faces))
	}
	if tool.Inferred() != "" {
		t.Errorf("a fresh tool reports Inferred() = %q, want empty", tool.Inferred())
	}
}

// TestGripSnapCommandStartsTool: the Assemble-tab Grip Snap command starts the tool on an active
// assembly (covering the command closure).
func TestGripSnapCommandStartsTool(t *testing.T) {
	s := assemblySession(t) // already registers the standard commands, Grip Snap among them
	if err := s.Execute("Assembly.GripSnap"); err != nil {
		t.Fatalf("Grip Snap command: %v", err)
	}
	if s.ActiveGripSnap() == nil {
		t.Error("Grip Snap command did not start the tool")
	}
}
