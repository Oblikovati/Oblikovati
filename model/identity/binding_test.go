// SPDX-License-Identifier: GPL-2.0-only

package identity

import (
	"testing"

	"oblikovati.org/math"
)

// These tests exercise fallbackMatch directly — the live degraded-binding tiers
// (M31-F06) the model reaches through [RecoverLost] on an exact miss. They build a
// key with keyFor (capturing the parent/anchor hints) and hand fallbackMatch the
// SURVIVING entities, modelling "the exact entity is gone after an upstream edit".

// tokenLineage is a "/"-delimited lineage (parent = all but the last token) that
// exposes its parent for ancestral matching, modelling kernel/topo lineage keys.
type tokenLineage struct {
	key    string
	parent string // "" means a root with no parent
}

func (l tokenLineage) LineageKey() []byte { return []byte(l.key) }
func (l tokenLineage) ParentKey() []byte {
	if l.parent == "" {
		return nil
	}
	return []byte(l.parent)
}

// kinNode is an entity with a token lineage and an optional geometric anchor.
type kinNode struct {
	kind    EntityKind
	lin     tokenLineage
	at      math.Point3
	located bool
}

func (e kinNode) EntityKind() EntityKind { return e.kind }
func (e kinNode) Lineage() Lineage       { return e.lin }
func (e kinNode) Anchor() (math.Point3, bool) {
	return e.at, e.located
}

// sibling builds a face whose lineage is parent/child, optionally anchored at p.
func sibling(parent, child string, p math.Point3, located bool) kinNode {
	return kinNode{
		kind:    KindFace,
		lin:     tokenLineage{key: parent + "/" + child, parent: parent},
		at:      p,
		located: located,
	}
}

// equidistantSiblings is two anchored ext#1 siblings, symmetric about the origin —
// the ambiguous case both the tie and no-anchor tests model.
func equidistantSiblings() []Entity {
	return []Entity{
		sibling("ext#1", "edge#7", math.P3(5, 0, 0), true),
		sibling("ext#1", "edge#8", math.P3(-5, 0, 0), true),
	}
}

// --- ancestral tier ---

func TestAncestralBindsLoneSurvivingSibling(t *testing.T) {
	keyed := sibling("ext#1", "edge#3", math.P3(0, 0, 0), false)
	key := keyFor(0, keyed)

	// The exact edge#3 is gone; one sibling under ext#1 survives (renamed edge#9).
	got, match := fallbackMatch(key, []Entity{sibling("ext#1", "edge#9", math.P3(0, 0, 0), false)})
	if match != MatchAncestral {
		t.Fatalf("match = %v, want ancestral", match)
	}
	if string(got.Lineage().LineageKey()) != "ext#1/edge#9" {
		t.Errorf("bound to %q, want ext#1/edge#9", got.Lineage().LineageKey())
	}
}

func TestAncestralMissWhenNoSiblingSharesParent(t *testing.T) {
	key := keyFor(0, sibling("ext#1", "edge#3", math.P3(0, 0, 0), false))

	// Survivor belongs to a different parent — not a sibling, so the reference is lost.
	if _, match := fallbackMatch(key, []Entity{sibling("ext#2", "edge#3", math.P3(0, 0, 0), false)}); match != MatchNone {
		t.Fatalf("match = %v, want none", match)
	}
}

func TestRootLineageHasNoAncestralFallback(t *testing.T) {
	key := keyFor(0, kinNode{kind: KindFace, lin: tokenLineage{key: "base", parent: ""}})

	survivors := []Entity{kinNode{kind: KindFace, lin: tokenLineage{key: "base2", parent: ""}}}
	if _, match := fallbackMatch(key, survivors); match != MatchNone {
		t.Fatalf("match = %v, want none (root has no parent to recover via)", match)
	}
}

func TestAncestralSiblingMustMatchKind(t *testing.T) {
	key := keyFor(0, sibling("ext#1", "edge#3", math.P3(0, 0, 0), false))

	// Same parent, but a vertex — not a candidate for an edge key.
	survivors := []Entity{kinNode{kind: KindVertex, lin: tokenLineage{key: "ext#1/vert#0", parent: "ext#1"}}}
	if _, match := fallbackMatch(key, survivors); match != MatchNone {
		t.Fatalf("match = %v, want none (kind must match)", match)
	}
}

// --- geometric tier ---

func TestGeometricPicksNearestAmbiguousSibling(t *testing.T) {
	key := keyFor(0, sibling("ext#1", "edge#3", math.P3(10, 0, 0), true))

	// Two siblings survive — disambiguate by nearness to the mint anchor.
	far := sibling("ext#1", "edge#7", math.P3(-10, 0, 0), true)
	near := sibling("ext#1", "edge#8", math.P3(9, 0, 0), true)
	got, match := fallbackMatch(key, []Entity{far, near})
	if match != MatchGeometric {
		t.Fatalf("match = %v, want geometric", match)
	}
	if string(got.Lineage().LineageKey()) != "ext#1/edge#8" {
		t.Errorf("bound to %q, want the nearer ext#1/edge#8", got.Lineage().LineageKey())
	}
}

func TestGeometricMissOnEquidistantTie(t *testing.T) {
	key := keyFor(0, sibling("ext#1", "edge#3", math.P3(0, 0, 0), true))

	// Two equidistant siblings — no defensible winner.
	if _, match := fallbackMatch(key, equidistantSiblings()); match != MatchNone {
		t.Fatalf("match = %v, want none (geometric tie is not defensible)", match)
	}
}

func TestGeometricMissWithoutAnchor(t *testing.T) {
	key := keyFor(0, sibling("ext#1", "edge#3", math.P3(0, 0, 0), false)) // no anchor captured

	if _, match := fallbackMatch(key, equidistantSiblings()); match != MatchNone {
		t.Fatalf("match = %v, want none (no anchor to disambiguate)", match)
	}
}

// --- match-type ordering and strings ---

func TestMatchTypeStringAndOrdering(t *testing.T) {
	if MatchExact <= MatchAncestral || MatchAncestral <= MatchGeometric || MatchGeometric <= MatchNone {
		t.Fatalf("match quality must ascend none<geometric<ancestral<exact")
	}
	for m, want := range map[MatchType]string{
		MatchExact: "exact", MatchAncestral: "ancestral", MatchGeometric: "geometric", MatchNone: "none",
	} {
		if got := m.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", m, got, want)
		}
	}
}
