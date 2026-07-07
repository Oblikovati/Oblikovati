// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// TestFilletOpenCurvedTangentChainErrors is the ADR-0050 P6 partial-result contract: filleting an
// OPEN tangent chain that crosses a curved face (a straight-arc-straight sub-selection of the top
// rim) is not yet buildable — its setback end-caps are future work. It must fail with an ACTIONABLE
// message (select the whole loop, or one edge at a time), not the misleading "miter corner's outer
// face must be planar" the per-edge path emitted, and not a silently wrong body.
func TestFilletOpenCurvedTangentChainErrors(t *testing.T) {
	box := csgBox(math.P3(0, 0, 0), 4, 4, 4)
	var verts [][]byte
	for _, e := range box.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.X == c.X && a.Y == c.Y {
			verts = append(verts, e.ReferenceKey())
		}
	}
	filleted, err := ops.FilletEdges(box, verts, 0.5)
	if err != nil {
		t.Fatalf("vertical fillet setup: %v", err)
	}
	var seed []byte
	maxZ := 0.0
	for _, v := range filleted.Vertices() {
		if v.Point().Z > maxZ {
			maxZ = v.Point().Z
		}
	}
	for _, e := range filleted.Edges() {
		if e.StartVertex().Point().Z >= maxZ-1e-9 && e.EndVertex().Point().Z >= maxZ-1e-9 {
			if _, isArc := e.Geometry().(geom.Arc3d); !isArc {
				seed = e.ReferenceKey()
				break
			}
		}
	}
	chain, _, err := ops.TangentEdgeChain(filleted, seed, ops.DefaultTangentChainAngle)
	if err != nil {
		t.Fatal(err)
	}
	open := chain[:3] // straight, arc, straight — an open sub-run of the closed loop

	_, err = ops.FilletEdges(filleted, open, 0.25)
	if err == nil {
		t.Fatal("open curved tangent chain must fail (end-caps unimplemented), got a body")
	}
	if strings.Contains(err.Error(), "miter") {
		t.Errorf("open-chain error should be actionable, not the miter message: %v", err)
	}
	if !strings.Contains(err.Error(), "OPEN tangent chain") {
		t.Errorf("expected an actionable open-chain message, got: %v", err)
	}
}
