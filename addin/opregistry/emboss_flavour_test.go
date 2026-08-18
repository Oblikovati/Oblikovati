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

// The emboss's authoring surface (#1893). Two of these options were UNREACHABLE rather than
// missing: EmbossDefinition already carried a Taper, and the op passed a hard-coded 0 for it. So
// these tests assert what the schema actually delivers into the definition, not merely that a
// request is accepted — the latter passed for the whole time the taper was being discarded.

// lastEmbossDef returns the definition of the part's most recently added emboss.
func lastEmbossDef(t *testing.T, s *app.Session) *feature.EmbossDefinition {
	t.Helper()
	fs := s.ActiveDocument().Content().(*compdef.PartComponentDefinition).Features()
	for i := fs.Count() - 1; i >= 0; i-- {
		if e, ok := fs.Item(i).Definition().(*feature.EmbossFeature); ok {
			return e.Definition()
		}
	}
	t.Fatal("no emboss feature on the part")
	return nil
}

// TestEmbossTaperReachesTheDefinition: the wall draft is the option the model had all along and the
// wire could not ask for. A request that merely succeeds proves nothing here, so the test reads the
// angle back off the definition (#1893).
func TestEmbossTaperReachesTheDefinition(t *testing.T) {
	s, _, _ := extrudedSolid(t)
	if _, err := applyMap(t, s, "emboss", map[string]any{
		"sketchIndex": 0, "depth": "1 mm", "taper": "10 deg",
	}); err != nil {
		t.Fatalf("emboss with a taper: %v", err)
	}
	if got := lastEmbossDef(t, s).Taper; got < 0.17 || got > 0.176 { // 10° = 0.1745 rad
		t.Errorf("emboss taper reached the definition as %g rad, want ≈0.1745 (10 deg)", got)
	}
}

// TestEmbossTypeReachesTheDefinition: each flavour spelling maps to its EmbossType.
func TestEmbossTypeReachesTheDefinition(t *testing.T) {
	for _, c := range []struct {
		spelling string
		want     feature.EmbossType
	}{
		{"fromFace", feature.EmbossFromFace},
		{"engraveFromFace", feature.EngraveFromFace},
		{"fromPlane", feature.EmbossEngraveFromPlane},
	} {
		t.Run(c.spelling, func(t *testing.T) {
			s, _, _ := extrudedSolid(t)
			if _, err := applyMap(t, s, "emboss", map[string]any{
				"sketchIndex": 0, "depth": "1 mm", "type": c.spelling,
			}); err != nil {
				t.Fatalf("emboss type %q: %v", c.spelling, err)
			}
			if got := lastEmbossDef(t, s).Type; got != c.want {
				t.Errorf("type %q reached the definition as %v, want %v", c.spelling, got, c.want)
			}
		})
	}
}

// TestEmbossTypeRejectsAContradictoryEngrave: the older `engrave` shorthand and an explicit
// non-engraving type disagree, and honouring either would invert the other's intent.
func TestEmbossTypeRejectsAContradictoryEngrave(t *testing.T) {
	for _, spelling := range []string{"fromFace", "fromPlane"} {
		s, _, _ := extrudedSolid(t)
		if _, err := applyMap(t, s, "emboss", map[string]any{
			"sketchIndex": 0, "depth": "1 mm", "type": spelling, "engrave": true,
		}); err == nil {
			t.Errorf("type %q with engrave=true should be refused, not silently resolved", spelling)
		}
	}
}

// TestEmbossEngraveShorthandStillWorks: the shipped two-valued spelling keeps working on its own.
func TestEmbossEngraveShorthandStillWorks(t *testing.T) {
	s, _, _ := extrudedSolid(t)
	if _, err := applyMap(t, s, "emboss", map[string]any{
		"sketchIndex": 0, "depth": "1 mm", "engrave": true,
	}); err != nil {
		t.Fatalf("emboss with the engrave shorthand: %v", err)
	}
	if got := lastEmbossDef(t, s).Type; got != feature.EngraveFromFace {
		t.Errorf("engrave=true reached the definition as %v, want EngraveFromFace", got)
	}
}

// TestEmbossUnknownTypeIsAnError: an unrecognised flavour is refused rather than defaulting to a
// raise, which would be the opposite of an engrave the caller misspelled.
func TestEmbossUnknownTypeIsAnError(t *testing.T) {
	s, _, _ := extrudedSolid(t)
	if _, err := applyMap(t, s, "emboss", map[string]any{
		"sketchIndex": 0, "depth": "1 mm", "type": "raised",
	}); err == nil {
		t.Error("an unknown emboss type should error")
	}
}

// TestEmbossWrapToFaceReachesTheDefinition: the wrap face key is carried onto the definition, which
// is what lets the model bind and validate it. The seed part is a prism, so this wrap cannot succeed
// geometrically — what is under test is that the key arrives AND that the resulting refusal comes
// from the model looking at the face, which is only possible if it got there.
func TestEmbossWrapToFaceReachesTheDefinition(t *testing.T) {
	s, _, face := extrudedSolid(t)
	out, err := applyMap(t, s, "emboss", map[string]any{
		"sketchIndex": 0, "depth": "1 mm", "wrapToFace": face,
	})
	if err != nil {
		t.Fatalf("emboss with wrapToFace: %v", err)
	}
	if got := string(lastEmbossDef(t, s).WrapFaceKey); got != face {
		t.Errorf("wrapToFace reached the definition as %q, want %q", got, face)
	}
	var res struct {
		Healthy bool   `json:"healthy"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("decode reply: %v", err)
	}
	if res.Healthy {
		t.Error("wrapping onto a planar face should report unhealthy: a planar face needs no wrap")
	}
	if !strings.Contains(res.Reason, "cylindrical face") {
		t.Errorf("reason = %q, want it to name the cylindrical face the wrap needs", res.Reason)
	}
}
