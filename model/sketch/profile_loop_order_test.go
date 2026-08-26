// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"
	"testing"

	"oblikovati.org/math"
)

// TestMergeAbuttingLoopsOrderIsDeterministic pins #23: merging abutting hole cells must produce the
// same point SEQUENCE every call, not merely the same set of points.
//
// The merge chains surviving half-edges into rings, and both the adjacency build and the ring's
// start vertex used to come out of a Go map — whose iteration order is randomized. A ring that
// starts at a different vertex is the same hole geometrically, so every set-wise or sorted check
// passed, but an extrude builds ONE PRISM FACE PER POLYGON EDGE in this order: the hole prism came
// back with identical vertices on differently-oriented planes, and the boolean consuming it drifted.
// A real part (ReelToReel BigChunkyPlate) alternated between solid and surface with its volume
// wandering ~300 mm^3 across identical translations.
func TestMergeAbuttingLoopsOrderIsDeterministic(t *testing.T) {
	// two unit squares sharing the edge x=1: their union outline is one 0..2 x 0..1 ring.
	left := Loop{closed: true, polygon: []math.Point2{
		math.P2(0, 0), math.P2(1, 0), math.P2(1, 1), math.P2(0, 1)}}
	right := Loop{closed: true, polygon: []math.Point2{
		math.P2(1, 0), math.P2(2, 0), math.P2(2, 1), math.P2(1, 1)}}
	var want string
	for i := range 25 {
		got := loopSeqKey(mergeAbuttingLoops([]Loop{left, right}))
		if i == 0 {
			want = got
			continue
		}
		if got != want {
			t.Fatalf("run %d merged abutting loops in a different order:\n got %s\nwant %s", i, got, want)
		}
	}
}

// loopSeqKey renders loops as their ordered point sequences — order is the point, so it must not
// sort.
func loopSeqKey(ls []Loop) string {
	out := ""
	for _, l := range ls {
		out += "["
		for _, p := range l.Polygon() {
			out += fmt.Sprintf("%.3f,%.3f;", float64(p.X), float64(p.Y))
		}
		out += "]"
	}
	return out
}
