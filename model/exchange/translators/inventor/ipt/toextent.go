// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
	"math"
)

// The "To <face>" extent's TARGET. An extrude that terminates at a face carries no usable length —
// its dimLength1 is a stale leftover — so building it as a Dimension puts the feature wherever that
// number happens to point. BigChunkyPlate has ELEVEN of them (of 46 extent-bearing features), and
// one alone (a join on the z=1.8 plane whose stale dimLength1 read 2.6, running to z=-0.8) made the
// plate 3.80 cm thick against Inventor's own 3.00, which cost it every feature to the body gate.
//
// The GPL reference does NOT implement this: Create_FxExtrude_New special-cases only extend==5
// (ALL) and otherwise feeds dimLength1 straight to the extrusion, so it mis-builds a To extrude the
// same way we did. The target was found by differential instead — of the 31 Dimension and 11 To
// features in BigChunkyPlate, exactly two properties separate them, present on every To and no
// Dimension: 0x0d (a 3D8924FD boolean, the "is a plane" flag the reference names isToPlane on a
// HOLE) and 0x12, whose type the reference comments only as "???".

// toExtentTargetProp is the To extent's target, and toTargetNodeType its node type (the reference's
// Read_151280F0). Present on every To extrude and on no other kind — see the differential above.
const (
	toExtentTargetProp = 0x12
	toTargetNodeType   = 0x151280F0
)

// extentTo is the PartFeatureExtentEnum's "To" (Read_92637D29's 7th name).
const extentTo = 7

// toTargetTailBytes is the target frame's size: Read_151280F0 ends with three 3D f64 vectors and
// returns, so they are the payload's last 72 bytes — which is why the conditional u32 earlier in
// that node (present only when its count is 1) cannot shift them.
const toTargetTailBytes = 72

// toTargetPlane returns the plane a To extrude runs to, or ok=false when the file states none this
// layout can read (the caller then keeps the extrude's stated length rather than inventing a target).
//
// The frame is stored the same way a sketch placement is — origin, then two in-plane axes — so the
// normal is xAxis × yAxis. The reference's field names ('pos', 'dir', 'point') are misleading: the
// third vector reads (0,1,0)/(0,0,1) on real parts, i.e. a second AXIS, not a point.
//
// Self-validating on the corpus, which is what pins the offset: |xAxis| = 1.000000 on all 11 of
// BigChunkyPlate's targets, every target normal comes out PARALLEL to its own extrude's direction
// (as a To target must be, or the extrude could never reach it), and every target plane lands INSIDE
// the part's 3 cm thickness — as a face of the part must.
func toTargetPlane(nodes []dcNode, props []int) (SketchPlacement, bool) {
	if len(props) <= toExtentTargetProp {
		return SketchPlacement{}, false
	}
	n, ok := nodeAt(nodes, props[toExtentTargetProp])
	if !ok || n.typ != toTargetNodeType || len(n.payload) < toTargetTailBytes {
		return SketchPlacement{}, false
	}
	tail := n.payload[len(n.payload)-toTargetTailBytes:]
	p := SketchPlacement{Origin: vec3At(tail, 0), XAxis: vec3At(tail, 24), YAxis: vec3At(tail, 48)}
	if !isUnit(p.XAxis) || !isUnit(p.YAxis) {
		return SketchPlacement{}, false // a wrong offset yields arbitrary magnitudes, not 1
	}
	return p, true
}

// vec3At reads three f64 at off.
func vec3At(b []byte, off int) [3]float64 {
	return [3]float64{
		math.Float64frombits(binary.LittleEndian.Uint64(b[off:])),
		math.Float64frombits(binary.LittleEndian.Uint64(b[off+8:])),
		math.Float64frombits(binary.LittleEndian.Uint64(b[off+16:])),
	}
}

// isUnit reports whether v has unit length — the decode's own oracle for the target frame.
func isUnit(v [3]float64) bool {
	return math.Abs(math.Sqrt(v[0]*v[0]+v[1]*v[1]+v[2]*v[2])-1) <= 1e-9
}

// PlaneNormal returns the placement's normal, xAxis × yAxis (right-handed), matching sketch.NewPlane.
func (p SketchPlacement) PlaneNormal() [3]float64 {
	x, y := p.XAxis, p.YAxis
	return [3]float64{x[1]*y[2] - x[2]*y[1], x[2]*y[0] - x[0]*y[2], x[0]*y[1] - x[1]*y[0]}
}

// extrudeExtentEnum returns the feature's PartFeatureExtentEnum value.
func extrudeExtentEnum(nodes []dcNode, props []int) (uint16, bool) {
	if len(props) <= propExtent {
		return 0, false
	}
	n, ok := nodeAt(nodes, props[propExtent])
	if !ok || n.typ != extentEnumNodeType || len(n.payload) < enumValueOffset+2 {
		return 0, false
	}
	return binary.LittleEndian.Uint16(n.payload[enumValueOffset:]), true
}
