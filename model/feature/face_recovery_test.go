// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/identity"
)

// faceKeyFor renders a two-token face lineage (parent "grp:f#0", a given last step) as the
// reference key a feature would have stored when the user picked that face. 0x03 = topo.KindFace
// (the kernel reference-key kind byte, distinct from identity.KindFace).
func faceKeyFor(lastIndex int) []byte {
	lin := topo.NewLineage(topo.Tok("grp", "f", 0), topo.Tok("split", "side", lastIndex))
	return append([]byte{0x03}, lin.Key()...)
}

// siblingFaceBody builds coplanar triangles, one face each. The "keep" face carries a TWO-token
// lineage sharing parent "grp:f#0" with the given last step; a final unrelated face is single-token
// (no parent). extraSibling adds a SECOND face under the same parent (a different last step) so the
// parent has two surviving siblings — the case ancestral recovery must refuse without an anchor.
func siblingFaceBody(t *testing.T, lastIndex int, extraSibling bool) (*topo.Body, *topo.Face) {
	t.Helper()
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("f", "body", 0)))
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	id := 0
	tri := func(x float64, lin topo.Lineage) *topo.Face {
		a := bld.AddVertex(math.P3(x, 0, 0), topo.NewLineage(topo.Tok("f", "vertex", id)))
		b := bld.AddVertex(math.P3(x+1, 0, 0), topo.NewLineage(topo.Tok("f", "vertex", id+1)))
		c := bld.AddVertex(math.P3(x, 1, 0), topo.NewLineage(topo.Tok("f", "vertex", id+2)))
		ab := bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, topo.NewLineage(topo.Tok("f", "edge", id)))
		bc := bld.AddEdge(geom.NewLineSegment(b.Point(), c.Point()), b, c, topo.NewLineage(topo.Tok("f", "edge", id+1)))
		ca := bld.AddEdge(geom.NewLineSegment(c.Point(), a.Point()), c, a, topo.NewLineage(topo.Tok("f", "edge", id+2)))
		id += 3
		return bld.AddFace(pl, lin, topo.OuterLoop(topo.Fwd(ab), topo.Fwd(bc), topo.Fwd(ca)))
	}
	keep := tri(0, topo.NewLineage(topo.Tok("grp", "f", 0), topo.Tok("split", "side", lastIndex)))
	if extraSibling {
		tri(3, topo.NewLineage(topo.Tok("grp", "f", 0), topo.Tok("split", "side", lastIndex+1)))
	}
	tri(6, topo.NewLineage(topo.Tok("f", "face", 1))) // unrelated, single-token (no parent)
	return bld.Build(), keep
}

// TestResolveFacesHealsLostReferenceToAncestralSibling is the face counterpart of the ADR-0043 P6
// headline: a stored face key whose EXACT entity is gone but whose PARENT lineage still names
// exactly one surviving face is recovered ancestrally — bound and reported as a heal — not Sick.
func TestResolveFacesHealsLostReferenceToAncestralSibling(t *testing.T) {
	body, keep := siblingFaceBody(t, 7, false)
	lost := faceKeyFor(2) // same parent grp:f#0, a last step the body no longer has

	faces, heals, err := resolveFaces(body, [][]byte{lost}, nil)
	if err != nil {
		t.Fatalf("a recoverable face reference must heal, not error: %v", err)
	}
	if len(faces) != 1 || faces[0] != keep {
		t.Fatalf("heal bound the wrong face: got %v, want the lone parent sibling", faces)
	}
	if len(heals) != 1 || heals[0].Match != identity.MatchAncestral {
		t.Fatalf("expected one ancestral heal, got %+v", heals)
	}
}

// TestResolveFacesRefusesAmbiguousSiblingsWithoutAnchor pins the honesty invariant for faces: two
// surviving same-parent siblings with no mint-time anchor cannot be disambiguated, so the reference
// stays lost rather than binding a guess.
func TestResolveFacesRefusesAmbiguousSiblingsWithoutAnchor(t *testing.T) {
	body, _ := siblingFaceBody(t, 7, true) // two faces now share parent grp:f#0
	lost := faceKeyFor(2)

	if _, _, err := resolveFaces(body, [][]byte{lost}, nil); err == nil {
		t.Fatal("two same-parent face siblings with no anchor must NOT heal — recovery guessed")
	} else if !strings.Contains(err.Error(), "lost") {
		t.Errorf("error %q should report the reference as lost", err)
	}
}

// TestResolveFacesHealsAmbiguousSiblingsByAnchor is the face geometric tier: several surviving
// siblings are disambiguated by nearness to the mint-time anchor (the face centroid), recovering
// the face the user originally picked instead of staying lost.
func TestResolveFacesHealsAmbiguousSiblingsByAnchor(t *testing.T) {
	body, keep := siblingFaceBody(t, 7, true)
	lost := faceKeyFor(2)
	anchors := map[string]math.Point3{string(lost): topo.DescribeFace(keep).Centroid}

	faces, heals, err := resolveFaces(body, [][]byte{lost}, anchors)
	if err != nil {
		t.Fatalf("geometric recovery should heal an anchored ambiguous face reference: %v", err)
	}
	if len(faces) != 1 || faces[0] != keep {
		t.Fatalf("anchor near keep must bind keep, got %v", faces)
	}
	if len(heals) != 1 || heals[0].Match != identity.MatchGeometric {
		t.Fatalf("expected one geometric heal, got %+v", heals)
	}
}

// TestResolveFacesExactMatchNoHeal confirms an exactly-resolving key binds cleanly with no heal.
func TestResolveFacesExactMatchNoHeal(t *testing.T) {
	body, keep := siblingFaceBody(t, 7, false)

	faces, heals, err := resolveFaces(body, [][]byte{keep.ReferenceKey()}, nil)
	if err != nil || len(faces) != 1 || faces[0] != keep {
		t.Fatalf("exact key must bind itself with no error: faces=%v err=%v", faces, err)
	}
	if len(heals) != 0 {
		t.Errorf("an exact match must not report a heal, got %+v", heals)
	}
}

// dupFaceBody builds a body where two distinct faces share ONE lineage key — a topological-naming
// collision — plus an unrelated third face, to exercise the P0 ambiguity guard on the face path.
func dupFaceBody(t *testing.T) (*topo.Body, []byte) {
	t.Helper()
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("f", "body", 0)))
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	id := 0
	dup := topo.NewLineage(topo.Tok("f", "face", 0))
	tri := func(x float64, lin topo.Lineage) *topo.Face {
		a := bld.AddVertex(math.P3(x, 0, 0), topo.NewLineage(topo.Tok("f", "vertex", id)))
		b := bld.AddVertex(math.P3(x+1, 0, 0), topo.NewLineage(topo.Tok("f", "vertex", id+1)))
		c := bld.AddVertex(math.P3(x, 1, 0), topo.NewLineage(topo.Tok("f", "vertex", id+2)))
		ab := bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, topo.NewLineage(topo.Tok("f", "edge", id)))
		bc := bld.AddEdge(geom.NewLineSegment(b.Point(), c.Point()), b, c, topo.NewLineage(topo.Tok("f", "edge", id+1)))
		ca := bld.AddEdge(geom.NewLineSegment(c.Point(), a.Point()), c, a, topo.NewLineage(topo.Tok("f", "edge", id+2)))
		id += 3
		return bld.AddFace(pl, lin, topo.OuterLoop(topo.Fwd(ab), topo.Fwd(bc), topo.Fwd(ca)))
	}
	tri(0, dup)
	tri(3, dup) // second face, same lineage ⇒ same key
	tri(6, topo.NewLineage(topo.Tok("f", "face", 1)))
	body := bld.Build()
	return body, body.Faces()[0].ReferenceKey()
}

// TestBindFaceCollisionIsHonestError pins that a key matching MORE THAN ONE face, with no surviving
// sibling to recover to, is an honest error (the ADR-0043 P0 guard) — never a silent first-match.
func TestBindFaceCollisionIsHonestError(t *testing.T) {
	body, colliding := dupFaceBody(t)

	if _, _, err := bindFace(body, colliding, nil); err == nil {
		t.Fatal("a colliding face key must error, not silently bind the first match")
	}
}

// TestFindOrRecoverFaceAncestralSilent covers the silent (no-heal-channel) helper: it recovers a
// lone ancestral sibling for a best-effort caller, returning the face and true.
func TestFindOrRecoverFaceAncestralSilent(t *testing.T) {
	body, keep := siblingFaceBody(t, 7, false)
	lost := faceKeyFor(2)

	got, ok := FindOrRecoverFace(body, lost)
	if !ok || got != keep {
		t.Fatalf("FindOrRecoverFace should recover the lone ancestral sibling, got (%v,%v)", got, ok)
	}
	// With two same-parent siblings and no anchor, a silent caller must NOT guess.
	ambiguous, _ := siblingFaceBody(t, 7, true)
	if _, ok := FindOrRecoverFace(ambiguous, lost); ok {
		t.Error("FindOrRecoverFace must not guess between two anchorless siblings")
	}
}
