// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// keyString returns an edge reference key as a readable string (the \x02 kind byte stripped).
func keyString1536(k []byte) string {
	if len(k) > 0 && k[0] < 0x20 {
		return string(k[1:])
	}
	return string(k)
}

// TestRepro1536SecondFilletEdgeKey instruments the #1536 scenario: a first fillet on
// Extrusion1:side-edge#2, then a second fillet on the OPPOSITE vertical edge whose key was
// captured from the displayed (filleted) body. It reports (a) whether any fillet:e#N key collides
// onto more than one edge, and (b) whether the second fillet keeps the first fillet's geometry.
func TestRepro1536SecondFilletEdgeKey(t *testing.T) {
	poly := []math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4.89099465009204, Y: 5.1416568653588115}, {X: 0, Y: 3}}
	box := buildPrism(poly, sketch.XYPlane(), span{near: 0, far: 5}, 0, "Extrusion1")

	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(box)

	// First fillet on side-edge#2 (corner 2, the far-out corner), exactly as the report stored it.
	edge1 := edgeByKeySuffix1536(t, box, "Extrusion1:side-edge#2")
	f1 := NewDressUpFeatures(fs).AddFillet([][]byte{edge1}, func() float64 { return 1 })
	fs.Recompute()
	if !f1.Health().OK() {
		t.Fatalf("first fillet sick: %+v", f1.Health())
	}
	res1 := fs.Result()[0]
	if n := cylinderFaces1494(res1); n != 1 {
		t.Fatalf("after first fillet: %d cylinder faces, want 1", n)
	}

	// Report key collisions on the first-fillet body: any key bound to >1 edge is the #1536 hazard.
	collisions := map[string]int{}
	for _, e := range res1.Edges() {
		collisions[keyString1536(e.ReferenceKey())]++
	}
	dupes := 0
	for k, c := range collisions {
		if c > 1 {
			dupes++
			t.Logf("COLLISION: key %q is bound to %d edges", k, c)
		}
	}
	t.Logf("first-fillet body: %d edges, %d distinct keys, %d colliding keys", len(res1.Edges()), len(collisions), dupes)

	// What does fillet:e#13 (the reported second-fillet pick) resolve to, and is it the opposite
	// vertical edge or something on the first fillet?
	for _, e := range res1.Edges() {
		if keyString1536(e.ReferenceKey()) == "fillet:e#13" {
			a, b := e.StartVertex().Point(), e.EndVertex().Point()
			_, isLine := e.Geometry().(geom.LineSegment)
			t.Logf("fillet:e#13 → edge %v→%v (line=%v)", a, b, isLine)
		}
	}

	// Now stack the second fillet on the EXACT edge the report stored — fillet:e#13, captured from
	// the displayed (first-fillet) body — and recompute the whole tree.
	edge2 := edgeByKeySuffix1536(t, res1, "fillet:e#13")
	f2 := NewDressUpFeatures(fs).AddFillet([][]byte{edge2}, func() float64 { return 1 })
	fs.Recompute()

	if !f2.Health().OK() {
		t.Fatalf("second fillet sick: %+v", f2.Health())
	}
	res2 := fs.Result()[0]
	if r := ops.Validate(res2); !r.Valid || !res2.IsSolid() {
		t.Fatalf("after second fillet: not a valid solid: %+v", r.Issues)
	}
	if n := cylinderFaces1494(res2); n != 2 {
		t.Errorf("after second fillet: %d cylinder faces, want 2 (the first fillet was broken/re-used)", n)
	}
}

// TestRepro1536NamingStability probes whether the fillet's construction-order edge naming
// (assembleBody → Tok("fillet","e",len(c.edges))) is STABLE across an upstream edit — the core
// property a topological-naming system must guarantee. It builds the first fillet at several
// radii and reports which physical edge "fillet:e#13" maps to each time. If the same key names a
// different edge as the radius changes, a stored second-fillet reference silently rebinds to the
// wrong edge after any upstream edit — the systemic fragility behind #1536/#1537.
func TestRepro1536NamingStability(t *testing.T) {
	poly := []math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4.89099465009204, Y: 5.1416568653588115}, {X: 0, Y: 3}}
	foot := func(radius float64) (math.Point3, bool) {
		box := buildPrism(poly, sketch.XYPlane(), span{near: 0, far: 5}, 0, "Extrusion1")
		fs := NewPartFeatures(nil, nil)
		NewBaseFeatures(fs).AddBase(box)
		edge1 := edgeByKeySuffix1536(t, box, "Extrusion1:side-edge#2")
		NewDressUpFeatures(fs).AddFillet([][]byte{edge1}, func() float64 { return radius })
		fs.Recompute()
		for _, e := range fs.Result()[0].Edges() {
			if keyString1536(e.ReferenceKey()) == "fillet:e#13" {
				return e.StartVertex().Point(), true
			}
		}
		return math.Point3{}, false
	}

	var ref math.Point3
	for i, r := range []float64{0.5, 1.0, 1.5, 2.0} {
		p, ok := foot(r)
		if !ok {
			t.Logf("radius %.1f: no fillet:e#13", r)
			continue
		}
		t.Logf("radius %.1f: fillet:e#13 foot = %v", r, p)
		if i == 0 {
			ref = p
		} else if p.DistanceTo(ref) > 1e-6 {
			t.Errorf("UNSTABLE NAMING: fillet:e#13 names a DIFFERENT edge at radius %.1f (%v vs %v at 0.5) "+
				"— a stored reference silently rebinds after an upstream edit (#1536 systemic flaw)", r, p, ref)
		}
	}
}

// TestRepro1536PreviewResult drives the live-preview seam exactly as the fillet tool does: it
// builds the first fillet, then asks PreviewResult for a DRAFT second fillet on fillet:e#13 and
// checks the previewed body keeps the first fillet (2 cylinders) rather than dropping/re-cutting
// it. A wrong preview here is what shows the user TWO red wedges in #1536.
func TestRepro1536PreviewResult(t *testing.T) {
	poly := []math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4.89099465009204, Y: 5.1416568653588115}, {X: 0, Y: 3}}
	box := buildPrism(poly, sketch.XYPlane(), span{near: 0, far: 5}, 0, "Extrusion1")
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(box)
	edge1 := edgeByKeySuffix1536(t, box, "Extrusion1:side-edge#2")
	NewDressUpFeatures(fs).AddFillet([][]byte{edge1}, func() float64 { return 1 })
	fs.Recompute()
	base := fs.Result()[0]

	draft := &FilletFeature{def: &FilletDefinition{
		EdgeKeys: [][]byte{edgeByKeySuffix1536(t, base, "fillet:e#13")},
		Radius:   func() float64 { return 1 },
	}}
	preview, err := fs.PreviewResult(draft)
	if err != nil {
		t.Fatalf("PreviewResult: %v", err)
	}
	if n := cylinderFaces1494(preview[0]); n != 2 {
		t.Errorf("preview body has %d cylinder faces, want 2 (first fillet + previewed second)", n)
	}
}

// edgeByKeySuffix1536 returns the reference key of the body edge whose readable key equals want.
func edgeByKeySuffix1536(t *testing.T, b *topo.Body, want string) []byte {
	t.Helper()
	for _, e := range b.Edges() {
		if keyString1536(e.ReferenceKey()) == want {
			return e.ReferenceKey()
		}
	}
	t.Fatalf("no edge with key %q", want)
	return nil
}
