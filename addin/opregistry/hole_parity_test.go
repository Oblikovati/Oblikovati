// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"strings"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The hole's placements, seat/tap split, clearance sizing and terminations over the JSON API
// (#1861, #1862, #1863). The model tests pin the geometry; these pin that each option REACHES the
// definition — the failure mode these issues describe is an implemented feature no author can ask
// for (an "unreachable API"), so the assertions read the definition back.

// boxWithTopFace seeds the 4×3×1 box every hole case drills into, returning the session and its
// top face's reference key.
func boxWithTopFace(t *testing.T) (*app.Session, string) {
	t.Helper()
	s, _, _ := extrudedSolid(t)
	key, _ := boxTopFace(t, s)
	return s, key
}

// holeDefAfter applies the hole tool with the given args and returns the resulting definition.
func holeDefAfter(t *testing.T, s *app.Session, args map[string]any) *feature.HoleDefinition {
	t.Helper()
	raw, _ := json.Marshal(args)
	if _, err := apply(t, s, "hole", string(raw)); err != nil {
		t.Fatalf("hole %v: %v", args, err)
	}
	feats := s.ActiveDocument().Content().(*compdef.PartComponentDefinition).Features()
	return feats.Item(feats.Count() - 1).Definition().(*feature.HoleFeature).Definition()
}

// TestTappedCounterboreIsExpressible is the point of splitting the seat from the tap: before, the
// seat enum carried "tapped", so asking for a counterbore that is ALSO tapped was impossible.
func TestTappedCounterboreIsExpressible(t *testing.T) {
	t.Parallel()
	s, top := boxWithTopFace(t)
	def := holeDefAfter(t, s, map[string]any{
		"faceRef": top, "type": "counterbore", "diameter": "5 mm", "depth": "8 mm",
		"counterDiameter": "10 mm", "counterDepth": "3 mm",
		"tap": "tapped", "designation": "M5x0.8", "threadClass": "6H", "leftHanded": true,
	})
	if def.Type != feature.CounterboreHole {
		t.Errorf("seat = %v, want it to stay a counterbore", def.Type)
	}
	if !def.Tap.Tapped || def.Tap.Designation != "M5x0.8" || def.Tap.Class != "6H" || !def.Tap.LeftHanded {
		t.Errorf("tap = %+v, want a left-hand M5x0.8 class 6H tap on top of the seat", def.Tap)
	}
}

// TestLegacyTappedTypeStillMeansATappedHole: the older spelling must keep working, now as a drilled
// hole that is also tapped rather than as a seat of its own.
func TestLegacyTappedTypeStillMeansATappedHole(t *testing.T) {
	t.Parallel()
	s, top := boxWithTopFace(t)
	def := holeDefAfter(t, s, map[string]any{
		"faceRef": top, "type": "tapped", "diameter": "5 mm", "depth": "8 mm", "designation": "M6",
	})
	if def.Type != feature.DrilledHole || !def.Tap.Tapped || def.Tap.Designation != "M6" {
		t.Errorf("seat = %v tap = %+v, want a DRILLED seat that is tapped M6", def.Type, def.Tap)
	}
}

// TestSpotFaceSeatReachesTheDefinition: the seat that used to drop entirely on import.
func TestSpotFaceSeatReachesTheDefinition(t *testing.T) {
	t.Parallel()
	s, top := boxWithTopFace(t)
	def := holeDefAfter(t, s, map[string]any{
		"faceRef": top, "type": "spotface", "diameter": "5 mm", "depth": "8 mm",
		"counterDiameter": "12 mm", "counterDepth": "1 mm",
	})
	if def.Type != feature.SpotFaceHole {
		t.Errorf("seat = %v, want SpotFaceHole (not collapsed into a counterbore)", def.Type)
	}
}

// TestClearanceHoleCarriesItsFastener: the definition must keep the FASTENER, since that is what
// keeps the bore sized to it after an edit.
func TestClearanceHoleCarriesItsFastener(t *testing.T) {
	t.Parallel()
	s, top := boxWithTopFace(t)
	def := holeDefAfter(t, s, map[string]any{
		"faceRef": top, "depth": "8 mm",
		"clearance": map[string]any{"fastener": "M6", "fit": "free"},
	})
	if def.Clearance.Fastener != "M6" || def.Clearance.Fit != "free" {
		t.Errorf("clearance = %+v, want the M6 free-fit fastener recorded", def.Clearance)
	}
}

// TestHoleNeedsASize: a hole with neither a diameter nor a fastener has no size at all, and saying
// so here beats surfacing it later as "diameter 0 must be > 0".
func TestHoleNeedsASize(t *testing.T) {
	t.Parallel()
	s, top := boxWithTopFace(t)
	raw, _ := json.Marshal(map[string]any{"faceRef": top, "depth": "8 mm"})
	_, err := apply(t, s, "hole", string(raw))
	if err == nil || !strings.Contains(err.Error(), "diameter") {
		t.Errorf("a hole with no size gave %v; want an error naming diameter or clearance", err)
	}
}

// TestSketchPlacementReachesTheDefinition: the placement that turns one feature into many bores.
func TestSketchPlacementReachesTheDefinition(t *testing.T) {
	t.Parallel()
	s, top := boxWithTopFace(t)
	def := holeDefAfter(t, s, map[string]any{
		"faceRef": top, "diameter": "5 mm", "depth": "8 mm",
		"placement": "sketch", "placementSketchIndex": 0, "placementFlipped": true,
	})
	p, ok := def.Placement.(feature.SketchHolePlacement)
	if !ok || !p.Flipped {
		t.Errorf("placement = %#v, want a flipped SketchHolePlacement", def.Placement)
	}
}

// TestUnknownPlacementIsRejected keeps a typo from degrading to the face placement, which would
// drill one hole at a centroid and look like the feature "worked".
func TestUnknownPlacementIsRejected(t *testing.T) {
	t.Parallel()
	s, top := boxWithTopFace(t)
	raw, _ := json.Marshal(map[string]any{
		"faceRef": top, "diameter": "5 mm", "depth": "8 mm", "placement": "onEdge",
	})
	if _, err := apply(t, s, "hole", string(raw)); err == nil {
		t.Error("hole accepted placement \"onEdge\"; an unknown rule must not fall back to the face placement")
	}
}

// TestHoleTerminationReachesTheDefinition wires the to-face stop through to the model.
func TestHoleTerminationReachesTheDefinition(t *testing.T) {
	t.Parallel()
	s, top := boxWithTopFace(t)
	def := holeDefAfter(t, s, map[string]any{
		"faceRef": top, "diameter": "5 mm",
		"termination": "to-face", "toFace": "origin/plane/xy",
	})
	if def.Termination != feature.ToFaceExtent || def.ToPlane == nil {
		t.Errorf("termination = %v with toPlane %v, want a bound to-face stop", def.Termination, def.ToPlane)
	}
}

// TestHoleToFaceNeedsATarget: an extent that stops on a face and is given none is a caller error.
func TestHoleToFaceNeedsATarget(t *testing.T) {
	t.Parallel()
	s, top := boxWithTopFace(t)
	raw, _ := json.Marshal(map[string]any{
		"faceRef": top, "diameter": "5 mm", "termination": "to-face",
	})
	_, err := apply(t, s, "hole", string(raw))
	if err == nil || !strings.Contains(err.Error(), "toFace") {
		t.Errorf("to-face hole with no target gave %v; want an error naming \"toFace\"", err)
	}
}
