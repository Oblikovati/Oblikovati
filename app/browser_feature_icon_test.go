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
	t.Parallel()
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
	t.Parallel()
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

// TestBrowserWorkAndSketchNodesCarryIcons: top-level sketches, 3D sketches, and the user/origin
// work planes, axes and points all carry their glyph keys on the browser node (#1264).
func TestBrowserWorkAndSketchNodesCarryIcons(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	def.Sketches().Add(sketch.XYPlane()) // top-level 2D sketch
	def.Sketches3D().Add()               // 3D sketch
	def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 2 })
	def.WorkAxes().AddByPlaneIntersection(feature.OriginXYPlane, feature.OriginXZPlane)
	def.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(1, 1, 1) })
	def.Recompute()

	root := BuildBrowser(s)
	wants := map[string]string{ // node kind -> expected icon key
		"sketch": "create-sketch", "sketch3d": "create-sketch-3d",
		"workplane": iconWorkPlane, "workaxis": iconWorkAxis, "workpoint": iconWorkPoint,
	}
	for kind, icon := range wants {
		n, ok := firstNodeOfKind(root, kind)
		if !ok {
			t.Errorf("no %q node in the browser", kind)
			continue
		}
		if n.Icon != icon {
			t.Errorf("%q node icon = %q, want %q", kind, n.Icon, icon)
		}
	}
}

// firstNodeOfKind returns the first node of the given kind that carries an icon, depth-first.
func firstNodeOfKind(n BrowserNode, kind string) (BrowserNode, bool) {
	if n.Kind == kind && n.Icon != "" {
		return n, true
	}
	for _, c := range n.Children {
		if got, ok := firstNodeOfKind(c, kind); ok {
			return got, true
		}
	}
	return BrowserNode{}, false
}
