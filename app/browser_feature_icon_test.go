// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// TestFeatureIconMapsKinds: known kinds map to their glyph, fillet variants collapse onto one,
// and an unknown kind falls back to the default so every feature row still carries an icon.
func TestFeatureIconMapsKinds(t *testing.T) {
	cases := map[string]string{
		"extrude": "extrude", "revolve": "revolve", "hole": "hole",
		"face-fillet": "fillet", "rule-fillet": "fillet", "chamfer": "chamfer",
	}
	for kind, want := range cases {
		if got := featureIcon(kind); got != want {
			t.Errorf("featureIcon(%q) = %q, want %q", kind, got, want)
		}
	}
	if got := featureIcon("totally-unknown-kind"); got == "" {
		t.Error("an unmapped kind should still get a default icon, got empty")
	}
}

// TestBrowserFeatureNodeCarriesIcon: a feature in the history gets its glyph key on the browser
// node so the head can draw it (#1264).
func TestBrowserFeatureNodeCarriesIcon(t *testing.T) {
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(0, 0))
	c1 := sk.Points().Add(math.P2(2, 0))
	c2 := sk.Points().Add(math.P2(2, 2))
	c3 := sk.Points().Add(math.P2(0, 2))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 3 })
	def.Recompute()

	node := findBrowserNode(BuildBrowser(s), "feature", "Extrusion1")
	if node == nil {
		t.Fatal("no Extrusion1 feature node in the browser tree")
	}
	if node.Icon != "extrude" {
		t.Errorf("extrude feature node Icon = %q, want %q", node.Icon, "extrude")
	}
}
