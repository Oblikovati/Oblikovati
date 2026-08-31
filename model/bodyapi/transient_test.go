// SPDX-License-Identifier: GPL-2.0-only

package bodyapi

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// The session's transient body factory and registry (M07-F05, #628): handles are positive, never
// reused, and every body the wire layer hands back is one of this registry's own.

// pt and vec build the contract's point/vector for the factory calls.
func pt(x, y, z float64) types.Point   { return types.NewPoint(x, y, z) }
func vec(x, y, z float64) types.Vector { return types.NewVector(x, y, z) }

// blockBody registers a 2×2×2 transient block and returns the registry with it.
func blockBody(t *testing.T) (*TransientBRep, contract.TransientBody) {
	t.Helper()
	reg := NewTransientBRep(ops.DefaultQuality())
	b, err := reg.CreateSolidBlock(pt(0, 0, 0), pt(2, 2, 2))
	if err != nil {
		t.Fatalf("CreateSolidBlock: %v", err)
	}
	return reg, b
}

// TestTransientPrimitivesCarryTheirTopology: each factory registers a solid whose counts and volume
// come from the kernel body, not from a cached number.
func TestTransientPrimitivesCarryTheirTopology(t *testing.T) {
	reg := NewTransientBRep(ops.DefaultQuality())
	block, err := reg.CreateSolidBlock(pt(0, 0, 0), pt(2, 2, 2))
	if err != nil {
		t.Fatalf("CreateSolidBlock: %v", err)
	}
	if _, err := reg.CreateSolidCylinderCone(pt(0, 0, 0), pt(0, 0, 4), 2, 2); err != nil {
		t.Fatalf("CreateSolidCylinderCone: %v", err)
	}
	if _, err := reg.CreateSolidSphere(pt(0, 0, 0), 2); err != nil {
		t.Fatalf("CreateSolidSphere: %v", err)
	}
	if _, err := reg.CreateSolidTorus(pt(0, 0, 0), vec(0, 0, 1), 4, 1); err != nil {
		t.Fatalf("CreateSolidTorus: %v", err)
	}
	tb := block.(*TransientBody)
	if !tb.IsSolid() || tb.FaceCount() != 6 || tb.EdgeCount() != 12 || tb.VertexCount() != 8 {
		t.Errorf("block reports solid=%v faces=%d edges=%d vertices=%d, want true/6/12/8",
			tb.IsSolid(), tb.FaceCount(), tb.EdgeCount(), tb.VertexCount())
	}
	if tb.ShellCount() != 1 || tb.WireCount() != 0 {
		t.Errorf("block reports %d shells / %d wires, want 1 / 0", tb.ShellCount(), tb.WireCount())
	}
	if stdmath.Abs(tb.Volume()-8) > 1e-9 {
		t.Errorf("block volume = %g, want 8", tb.Volume())
	}
	if got := reg.Handles(); len(got) != 4 || got[0] != tb.Handle() {
		t.Errorf("handles = %v, want the four registered bodies with the block first", got)
	}
}

// TestTransientRegistryHandleLifecycle: a handle resolves while it lives, Delete reports whether it
// existed, Replace swaps the body in place, and a deleted handle is never handed out again.
func TestTransientRegistryHandleLifecycle(t *testing.T) {
	reg, block := blockBody(t)
	h := block.(*TransientBody).Handle()
	if _, ok := reg.ByHandle(h); !ok {
		t.Fatalf("ByHandle(%d) missed a live body", h)
	}
	bigger, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(3, 3, 3), "replacement")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	if err := reg.Replace(h, bigger); err != nil {
		t.Fatalf("Replace: %v", err)
	}
	if got := block.(*TransientBody).Volume(); stdmath.Abs(got-27) > 1e-9 {
		t.Errorf("volume after Replace = %g, want the replacement's 27", got)
	}
	if err := reg.Replace(h+100, bigger); err == nil || !strings.Contains(err.Error(), "handle") {
		t.Errorf("Replace on an unknown handle = %v, want an error naming the handle", err)
	}
	if !reg.Delete(h) || reg.Delete(h) {
		t.Error("Delete must report true once and false for an already freed handle")
	}
	if _, ok := reg.ByHandle(h); ok {
		t.Error("ByHandle resolved a deleted handle")
	}
	adopted := reg.Adopt(bigger)
	if adopted.Handle() == h {
		t.Errorf("Adopt reused the freed handle %d; handles are never reused in a session", h)
	}
}

// TestTransientCopyIsIndependent: a copy is a new handle with its own body, so replacing the source
// leaves the copy untouched.
func TestTransientCopyIsIndependent(t *testing.T) {
	reg, block := blockBody(t)
	clone, err := reg.Copy(block)
	if err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if clone.(*TransientBody).Handle() == block.(*TransientBody).Handle() {
		t.Error("Copy returned the source's own handle")
	}
	if got := clone.(*TransientBody).Volume(); stdmath.Abs(got-8) > 1e-9 {
		t.Errorf("copy volume = %g, want the source's 8", got)
	}
	if _, err := reg.Copy(foreignBody{}); err == nil {
		t.Error("Copy accepted a body this registry never issued")
	}
}

// TestTransientTransformMovesTheBody: a translation moves the registered body in place and keeps its
// volume; a body from outside the registry is refused with its type named.
func TestTransientTransformMovesTheBody(t *testing.T) {
	reg, block := blockBody(t)
	m := types.IdentityMatrix()
	m.Cells[3] = 10 // row-major translation in x
	if err := reg.Transform(block, m); err != nil {
		t.Fatalf("Transform: %v", err)
	}
	box := block.(*TransientBody).Topo().RangeBox()
	if stdmath.Abs(float64(box.Min.X)-10) > 1e-9 {
		t.Errorf("translated block starts at x=%g, want 10", box.Min.X)
	}
	err := reg.Transform(foreignBody{}, types.IdentityMatrix())
	if err == nil || !strings.Contains(err.Error(), "bodyapi.foreignBody") {
		t.Errorf("Transform on a foreign body = %v, want an error naming the offending type", err)
	}
}

// TestTransientDoBooleanCombinesInPlace: the blank is modified in place, once per operation, and the
// wire enum maps onto the kernel operation.
func TestTransientDoBooleanCombinesInPlace(t *testing.T) {
	cases := []struct {
		op   types.BooleanType
		want float64
	}{
		{types.BooleanUnion, 15},
		{types.BooleanDifference, 7},
		{types.BooleanIntersect, 1},
	}
	for _, c := range cases {
		reg := NewTransientBRep(ops.DefaultQuality())
		blank, err := reg.CreateSolidBlock(pt(0, 0, 0), pt(2, 2, 2))
		if err != nil {
			t.Fatalf("CreateSolidBlock blank: %v", err)
		}
		tool, err := reg.CreateSolidBlock(pt(1, 1, 1), pt(3, 3, 3))
		if err != nil {
			t.Fatalf("CreateSolidBlock tool: %v", err)
		}
		if err := reg.DoBoolean(blank, tool, c.op); err != nil {
			t.Fatalf("DoBoolean(%v): %v", c.op, err)
		}
		if got := blank.(*TransientBody).Volume(); stdmath.Abs(got-c.want) > 1e-9 {
			t.Errorf("%v volume = %g, want %g", c.op, got, c.want)
		}
	}
}

// TestTransientDoBooleanRejectsForeignOperands: both operands must be this registry's own.
func TestTransientDoBooleanRejectsForeignOperands(t *testing.T) {
	reg, block := blockBody(t)
	if err := reg.DoBoolean(foreignBody{}, block, types.BooleanUnion); err == nil {
		t.Error("DoBoolean accepted a foreign blank")
	}
	if err := reg.DoBoolean(block, foreignBody{}, types.BooleanUnion); err == nil {
		t.Error("DoBoolean accepted a foreign tool")
	}
}

// TestTransientSectionWithPlane: sectioning a block mid-height registers a new body carrying the
// section as wires, leaving the source alone.
func TestTransientSectionWithPlane(t *testing.T) {
	reg, block := blockBody(t)
	sec, err := reg.CreateIntersectionWithPlane(block, pt(0, 0, 1), vec(0, 0, 1))
	if err != nil {
		t.Fatalf("CreateIntersectionWithPlane: %v", err)
	}
	if got := sec.(*TransientBody).WireCount(); got == 0 {
		t.Error("section body carries no wire")
	}
	if sec.(*TransientBody).Handle() == block.(*TransientBody).Handle() {
		t.Error("the section reused the source's handle")
	}
	if _, err := reg.CreateIntersectionWithPlane(foreignBody{}, pt(0, 0, 1), vec(0, 0, 1)); err == nil {
		t.Error("CreateIntersectionWithPlane accepted a foreign body")
	}
}

// TestTransientCreateFromDefinition: a valid graph registers a body; a graph with a decode issue
// registers nothing and reports the issue by path.
func TestTransientCreateFromDefinition(t *testing.T) {
	reg := NewTransientBRep(ops.DefaultQuality())
	body, issues, err := reg.CreateFromDefinition(squareDef())
	if err != nil || len(issues) > 0 || body == nil {
		t.Fatalf("CreateFromDefinition(valid) = (%v, %v, %v), want a body and no issue", body, issues, err)
	}
	bad := squareDef()
	bad.Faces[0].Surface.Kind = "hyperboloid"
	body, issues, err = reg.CreateFromDefinition(bad)
	if err != nil {
		t.Fatalf("CreateFromDefinition(invalid) errored instead of reporting issues: %v", err)
	}
	if body != nil || len(issues) == 0 || issues[0].Path != "faces[0]" {
		t.Errorf("invalid graph = (%v, %v), want no body and an issue at faces[0]", body, issues)
	}
	if got := len(reg.Handles()); got != 1 {
		t.Errorf("registry holds %d handles, want only the valid body's 1", got)
	}
}

// foreignBody is a contract body this registry never issued — the named-decline fixture.
type foreignBody struct{}

func (foreignBody) IsSolid() bool    { return false }
func (foreignBody) FaceCount() int   { return 0 }
func (foreignBody) EdgeCount() int   { return 0 }
func (foreignBody) VertexCount() int { return 0 }
func (foreignBody) ShellCount() int  { return 0 }
func (foreignBody) WireCount() int   { return 0 }
func (foreignBody) Volume() float64  { return 0 }
