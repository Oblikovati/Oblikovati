// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/health"
	"oblikovati.org/model/identity"
)

// edgeKeyFor renders a two-token edge lineage (parent "grp:e#0", a given last step) as the
// reference key a feature would have stored when the user picked that edge.
func edgeKeyFor(lastIndex int) []byte {
	lin := topo.NewLineage(topo.Tok("grp", "e", 0), topo.Tok("base", "side", lastIndex))
	return append([]byte{0x02}, lin.Key()...) // 0x02 = topo.KindEdge
}

// siblingTriBody builds a triangle whose edge "ab" carries a TWO-token lineage sharing the parent
// "grp:e#0" with the given last step; the other two edges are unrelated (single-token, no parent).
// extraSibling adds a SECOND edge under the same parent (a different last step) so the parent has
// two surviving siblings — the case ancestral recovery must refuse without a geometric anchor.
func siblingTriBody(t *testing.T, lastIndex int, extraSibling bool) (*topo.Body, *topo.Edge) {
	t.Helper()
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("f", "body", 0)))
	a := bld.AddVertex(math.P3(0, 0, 0), topo.NewLineage(topo.Tok("f", "vertex", 0)))
	b := bld.AddVertex(math.P3(1, 0, 0), topo.NewLineage(topo.Tok("f", "vertex", 1)))
	c := bld.AddVertex(math.P3(0, 1, 0), topo.NewLineage(topo.Tok("f", "vertex", 2)))
	abLin := topo.NewLineage(topo.Tok("grp", "e", 0), topo.Tok("base", "side", lastIndex))
	ab := bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, abLin)
	bcLin := topo.NewLineage(topo.Tok("f", "edge", 1))
	if extraSibling {
		bcLin = topo.NewLineage(topo.Tok("grp", "e", 0), topo.Tok("base", "side", lastIndex+1))
	}
	bc := bld.AddEdge(geom.NewLineSegment(b.Point(), c.Point()), b, c, bcLin)
	ca := bld.AddEdge(geom.NewLineSegment(c.Point(), a.Point()), c, a, topo.NewLineage(topo.Tok("f", "edge", 2)))
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	bld.AddFace(pl, topo.NewLineage(topo.Tok("f", "face", 0)), topo.OuterLoop(topo.Fwd(ab), topo.Fwd(bc), topo.Fwd(ca)))
	return bld.Build(), ab
}

// TestResolveEdgesHealsLostReferenceToAncestralSibling is the ADR-0043 P6 headline: a stored edge
// key whose EXACT entity is gone (its lineage drifted under an upstream edit) but whose PARENT
// lineage still names exactly one surviving edge is recovered ancestrally — bound to that sibling
// and reported as a heal — instead of going Sick. This is what turns the fillet-stack reference
// loss into an auto-healed warning.
func TestResolveEdgesHealsLostReferenceToAncestralSibling(t *testing.T) {
	t.Parallel()
	body, kept := siblingTriBody(t, 7, false)
	lost := edgeKeyFor(2) // same parent grp:e#0, a last step the body no longer has

	edges, heals, err := resolveEdges(body, [][]byte{lost}, nil)
	if err != nil {
		t.Fatalf("a recoverable reference must heal, not error: %v", err)
	}
	if len(edges) != 1 || edges[0] != kept {
		t.Fatalf("heal bound the wrong edge: got %v, want the lone parent sibling", edges)
	}
	if len(heals) != 1 || heals[0].Match != identity.MatchAncestral {
		t.Fatalf("expected one ancestral heal, got %+v", heals)
	}
}

// TestResolveEdgesRefusesAmbiguousSiblingsWithoutAnchor pins the honesty invariant: when the lost
// key's parent has MORE THAN ONE surviving sibling, ancestral recovery cannot pick one and — with
// no mint-time anchor yet (the geometric tier is P6b) — the reference stays lost rather than binding
// a guess. A wrong heal is worse than an honest Sick.
func TestResolveEdgesRefusesAmbiguousSiblingsWithoutAnchor(t *testing.T) {
	t.Parallel()
	body, _ := siblingTriBody(t, 7, true) // two edges now share parent grp:e#0
	lost := edgeKeyFor(2)

	_, _, err := resolveEdges(body, [][]byte{lost}, nil)
	if err == nil {
		t.Fatal("two same-parent siblings with no anchor must NOT heal — recovery guessed")
	}
	if !strings.Contains(err.Error(), "lost") {
		t.Errorf("error %q should report the reference as lost", err)
	}
}

// TestResolveEdgesHealsAmbiguousSiblingsByAnchor is the ADR-0043 P6b geometric tier: when the
// lost key's parent has several surviving siblings, the mint-time anchor disambiguates by
// nearness — recovering the sibling the user originally picked instead of staying lost (the P6a
// outcome). The anchor sits on edge "ab"'s midpoint, so recovery must bind ab, not its peer.
func TestResolveEdgesHealsAmbiguousSiblingsByAnchor(t *testing.T) {
	t.Parallel()
	body, ab := siblingTriBody(t, 7, true) // ab + bc both under parent grp:e#0
	lost := edgeKeyFor(2)
	anchors := map[string]math.Point3{string(lost): math.P3(0.5, 0, 0)} // ab's endpoint midpoint

	edges, heals, err := resolveEdges(body, [][]byte{lost}, anchors)
	if err != nil {
		t.Fatalf("geometric recovery should heal an anchored ambiguous reference: %v", err)
	}
	if len(edges) != 1 || edges[0] != ab {
		t.Fatalf("anchor near ab must bind ab, got %v", edges)
	}
	if len(heals) != 1 || heals[0].Match != identity.MatchGeometric {
		t.Fatalf("expected one geometric heal, got %+v", heals)
	}
}

// TestFilletEdgeAnchorsRoundTrip pins that a fillet's mint-time edge anchors survive a recipe
// round trip (ADR-0043 P6b) — without them, a reopened document could not use the geometric tier.
func TestFilletEdgeAnchorsRoundTrip(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	key := edgeKeyFor(7)
	anchors := map[string]math.Point3{string(key): math.P3(1.25, -2.5, 3)}
	NewDressUpFeatures(fs).addFillet(&FilletDefinition{
		EdgeKeys: [][]byte{key}, Radius: constFloat(0.5), EdgeAnchors: anchors,
	})

	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	got := fresh.Item(0).Definition().(*FilletFeature).Definition().EdgeAnchors
	if len(got) != 1 {
		t.Fatalf("edge anchors after round trip = %v, want one entry", got)
	}
	p, ok := got[string(key)]
	if !ok || !p.IsEqualTo(math.P3(1.25, -2.5, 3), 1e-9) {
		t.Errorf("anchor after round trip = %v (present=%v), want (1.25,-2.5,3)", p, ok)
	}
}

// healingFeature is a fake feature that rebuilds successfully but reports a reference heal, to pin
// the engine's classification of a heal.
type healingFeature struct{ heals []ReferenceHeal }

func (healingFeature) Kind() string { return "healtest" }
func (f healingFeature) Recompute(in Input) (Output, error) {
	return Output{Bodies: append(append([]*topo.Body(nil), in.Bodies...), makeBody()), Heals: f.heals}, nil
}

// TestHealedReferenceClassifiesAsWarningNotSick pins ADR-0043 P6's health mapping: a feature that
// rebuilt on a recovered reference is a Warning (the body is kept, the drift is surfaced), distinct
// from both a clean recompute and a Sick lost-reference failure.
func TestHealedReferenceClassifiesAsWarningNotSick(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	fs.Add(body()) // a base body to operate on
	pf := fs.Add(healingFeature{heals: []ReferenceHeal{{Key: edgeKeyFor(2), Match: identity.MatchAncestral}}})
	fs.Recompute()

	if got := pf.Health().Status; got != health.Warning {
		t.Fatalf("a healed reference must classify as Warning, got %v (%s)", got, pf.Health().Reason)
	}
	if !strings.Contains(pf.Health().Reason, "healed") {
		t.Errorf("the warning should explain the heal, got %q", pf.Health().Reason)
	}
	if len(fs.Result()) == 0 {
		t.Error("the rebuilt body must be kept — a heal is not a passthrough")
	}
}

// TestParentOfKeyDropsMostSpecificToken pins the parent derivation both tiers compare on: the
// parent of a reference key is its lineage with the final token removed, and a single-token (root)
// key has no parent (nil) — so it has no ancestral fallback.
func TestParentOfKeyDropsMostSpecificToken(t *testing.T) {
	t.Parallel()
	two := edgeKeyFor(5) // grp:e#0/base:side#5
	if got := string(parentOfKey(two)); got != "grp:e#0" {
		t.Errorf("parentOfKey(two-token) = %q, want grp:e#0", got)
	}
	root := append([]byte{0x02}, topo.NewLineage(topo.Tok("only", "e", 1)).Key()...)
	if got := parentOfKey(root); got != nil {
		t.Errorf("parentOfKey(root) = %q, want nil (no parent)", got)
	}
}
