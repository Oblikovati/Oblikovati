// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/compdef"
)

// topFaceOfBody returns the body's highest face (greatest range-box centre z), the planar cap an
// external author would name as a to-face target — by GEOMETRY here, not by a key it cannot mint.
func topFaceOfBody(b *topo.Body) *topo.Face {
	var best *topo.Face
	for _, f := range b.Faces() {
		if best == nil || f.RangeBox().Center().Z > best.RangeBox().Center().Z {
			best = f
		}
	}
	return best
}

// TestExtrudeToFaceGeom: terminating a to-face extrude at a body face named by GEOMETRY
// (centroid+normal) builds the same body as naming it by reference key — proof the geometric
// target resolves to the same stop plane. This is the extent counterpart of the hole's
// PlacementFaceGeom path; an exporter reading Inventor's kToExtent target (a planar face it has no
// Oblikovati key for) must reach the to-face extent this way.
func TestExtrudeToFaceGeom(t *testing.T) {
	t.Parallel()
	byGeom := seedToFaceVolume(t)

	def := byGeom.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	g := topo.DescribeFace(topFaceOfBody(def.SurfaceBodies().Item(0)))
	args, _ := json.Marshal(map[string]any{
		"sketchIndex": 1, "profileIndex": 0, "extent": "to-face", "operation": "new",
		"toFaceGeom": map[string]any{
			"centroid": []float64{float64(g.Centroid.X), float64(g.Centroid.Y), float64(g.Centroid.Z)},
			"normal":   []float64{float64(g.Normal.X), float64(g.Normal.Y), float64(g.Normal.Z)},
		},
	})
	out, err := apply(t, byGeom, "extrude", string(args))
	if err != nil {
		t.Fatalf("to-face-geom extrude: %v", err)
	}
	var res struct {
		Bodies  int  `json:"bodies"`
		Healthy bool `json:"healthy"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !res.Healthy || res.Bodies != 2 {
		t.Fatalf("to-face-geom result = %+v, want healthy with 2 bodies", res)
	}
}

// TestExtrudeToFaceGeomFlippedNormalBinds: a to-face target whose recorded normal has the opposite
// sign convention (Inventor vs Oblikovati disagree, as observed on hole placement faces) must still
// bind the same planar face — face identity does not depend on the normal sign.
func TestExtrudeToFaceGeomFlippedNormalBinds(t *testing.T) {
	t.Parallel()
	byGeom := seedToFaceVolume(t)
	def := byGeom.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	g := topo.DescribeFace(topFaceOfBody(def.SurfaceBodies().Item(0)))
	args, _ := json.Marshal(map[string]any{
		"sketchIndex": 1, "profileIndex": 0, "extent": "to-face", "operation": "new",
		"toFaceGeom": map[string]any{
			"centroid": []float64{float64(g.Centroid.X), float64(g.Centroid.Y), float64(g.Centroid.Z)},
			"normal":   []float64{-float64(g.Normal.X), -float64(g.Normal.Y), -float64(g.Normal.Z)},
		},
	})
	if _, err := apply(t, byGeom, "extrude", string(args)); err != nil {
		t.Fatalf("to-face-geom (flipped normal) extrude: %v", err)
	}
}

// TestExtrudeToFaceGeomNoMatchDegradesGracefully: a geometric target that matches no face on the
// body degrades to an UNHEALTHY feature (healthy:false with a clear reason), NOT a hard error that
// aborts the operation — the hole's lost-placement-face pattern. A batch author (the exporter,
// reading an under-built base whose target face never formed) can then flag the feature and keep
// emitting the rest of the part instead of failing the whole document.
func TestExtrudeToFaceGeomNoMatchDegradesGracefully(t *testing.T) {
	t.Parallel()
	byGeom := seedToFaceVolume(t)
	args, _ := json.Marshal(map[string]any{
		"sketchIndex": 1, "profileIndex": 0, "extent": "to-face", "operation": "new",
		"toFaceGeom": map[string]any{"centroid": []float64{999, 999, 999}, "normal": []float64{0, 0, 1}},
	})
	out, err := apply(t, byGeom, "extrude", string(args))
	if err != nil {
		t.Fatalf("unmatchable to-face-geom should degrade gracefully, not error: %v", err)
	}
	var res struct {
		Healthy bool   `json:"healthy"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if res.Healthy || res.Reason == "" {
		t.Errorf("unmatchable to-face-geom result = %+v, want healthy:false with a reason", res)
	}
}

// TestExtrudeToFaceGeomBindsByPlaneThrough covers the plane-through fallback: a target whose recorded
// point lies ON the stop face's plane but OFF its centroid (the common case — an exporter records a
// point-on-face, not the running-body centroid) misses the exact centroid match and binds via
// FindPlanarFaceThrough instead, still terminating the extrude at that plane.
func TestExtrudeToFaceGeomBindsByPlaneThrough(t *testing.T) {
	t.Parallel()
	byGeom := seedToFaceVolume(t)
	def := byGeom.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	g := topo.DescribeFace(topFaceOfBody(def.SurfaceBodies().Item(0)))
	// Shift the recorded centroid 1 cm along +X: still on the top face's plane and inside its 4×3
	// boundary, but far past the 1e-3 exact-centroid tolerance, so only the plane-through path binds.
	args, _ := json.Marshal(map[string]any{
		"sketchIndex": 1, "profileIndex": 0, "extent": "to-face", "operation": "new",
		"toFaceGeom": map[string]any{
			"centroid": []float64{float64(g.Centroid.X) + 1, float64(g.Centroid.Y), float64(g.Centroid.Z)},
			"normal":   []float64{float64(g.Normal.X), float64(g.Normal.Y), float64(g.Normal.Z)},
		},
	})
	out, err := apply(t, byGeom, "extrude", string(args))
	if err != nil {
		t.Fatalf("plane-through to-face-geom: %v", err)
	}
	var res struct {
		Bodies  int  `json:"bodies"`
		Healthy bool `json:"healthy"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !res.Healthy || res.Bodies != 2 {
		t.Fatalf("plane-through to-face result = %+v, want healthy with 2 bodies", res)
	}
}

// seedToFaceVolume builds the base body a to-face extrude terminates against (a 10 mm prism), the
// shared fixture for the geometric to-face tests.
func seedToFaceVolume(t *testing.T) *app.Session {
	t.Helper()
	s := profiledPart(t)
	if _, err := apply(t, s, "extrude", `{"sketchIndex":0,"distance":"10 mm"}`); err != nil {
		t.Fatalf("seed extrude: %v", err)
	}
	return s
}
