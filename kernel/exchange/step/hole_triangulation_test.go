// SPDX-License-Identifier: GPL-2.0-only

package step_test

import (
	stdmath "math"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestImportedHoleNotTriangulatedOver guards the full-circle seam fix: a planar face bordering a
// circular hole must not have any triangle covering the hole. The bug was a full-circle edge
// whose parametrization did not start at its seam vertex, so discretizeEdge snapped the
// endpoints onto a ring that started elsewhere — a self-touching ring that earcut filled with a
// stray "wedge" across the hole (visible as red/green stray triangles in the drilled box).
func TestImportedHoleNotTriangulatedOver(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join("testdata", "occ", "drilled_box.step"))
	if err != nil {
		t.Fatalf("read drilled_box.step: %v", err)
	}
	bodies, _, err := step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	covering := 0
	for _, b := range bodies {
		for _, f := range b.Faces() {
			if _, ok := f.Geometry().(geom.Plane); !ok {
				continue
			}
			covering += holeCoveringTriangles(f)
		}
	}
	if covering > 0 {
		t.Errorf("%d planar triangles cover a hole (want 0) — the seam ring is self-touching again", covering)
	}
}

// holeCoveringTriangles counts a planar face's mesh triangles whose centroid lies inside one of
// the face's hole loops — a triangle that should not exist (the hole interior must stay empty).
func holeCoveringTriangles(f *topo.Face) int {
	holes := faceHoleDiscs(f)
	if len(holes) == 0 {
		return 0
	}
	m := tessellate.TessellateFace(f, ops.DefaultQuality())
	n := 0
	for t := 0; t+2 < len(m.Indices); t += 3 {
		a, b, c := m.Positions[m.Indices[t]], m.Positions[m.Indices[t+1]], m.Positions[m.Indices[t+2]]
		mx := (float64(a.X) + float64(b.X) + float64(c.X)) / 3
		my := (float64(a.Y) + float64(b.Y) + float64(c.Y)) / 3
		for _, h := range holes {
			if stdmath.Hypot(mx-h.cx, my-h.cy) < h.r*0.85 { // 0.85 to avoid rim-adjacent false positives
				n++
				break
			}
		}
	}
	return n
}

type holeDisc struct{ cx, cy, r float64 }

// faceHoleDiscs approximates each inner (hole) loop by its centroid and radius, in the same XY
// the planar triangulation uses (the fixtures' holes lie in Z planes, so XY suffices here).
func faceHoleDiscs(f *topo.Face) []holeDisc {
	var out []holeDisc
	for _, l := range f.Loops() {
		if l.IsOuter() {
			continue
		}
		var pts []math.Point3
		for _, u := range l.EdgeUses() {
			pts = append(pts, tessellate.TessellateEdge(u.Edge(), ops.DefaultQuality())...)
		}
		if len(pts) < 3 {
			continue
		}
		var cx, cy float64
		for _, p := range pts {
			cx, cy = cx+float64(p.X), cy+float64(p.Y)
		}
		cx, cy = cx/float64(len(pts)), cy/float64(len(pts))
		var r float64
		for _, p := range pts {
			if d := stdmath.Hypot(float64(p.X)-cx, float64(p.Y)-cy); d > r {
				r = d
			}
		}
		out = append(out, holeDisc{cx, cy, r})
	}
	return out
}
