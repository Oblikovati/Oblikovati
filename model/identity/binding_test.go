// SPDX-License-Identifier: GPL-2.0-only

package identity

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/health"
)

// --- fakes exercising the optional fallback capabilities (M31-F06) ---

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

func mintKey(t *testing.T, m *KeyManager, ctx ContextID, e Entity) RefKey {
	t.Helper()
	k, err := m.GetReferenceKey(ctx, e)
	if err != nil {
		t.Fatalf("GetReferenceKey: %v", err)
	}
	return k
}

// mintFor stands up a manager holding entities, keys keyed (which must be one of
// them), and returns the manager, its mutable source, and the key — the common
// arrange step for every rebind test. Mutate src.entities to model the edit, then
// BindKeyToObject(key).
func mintFor(t *testing.T, keyed Entity, entities ...Entity) (*KeyManager, *fakeSource, RefKey) {
	t.Helper()
	m := NewKeyManager()
	src := &fakeSource{entities: entities}
	ctx := m.CreateKeyContext(src)
	return m, src, mintKey(t, m, ctx, keyed)
}

// --- ancestral tier ---

func TestAncestralBindsLoneSurvivingSibling(t *testing.T) {
	keyed := sibling("ext#1", "edge#3", math.P3(0, 0, 0), false)
	m, src, key := mintFor(t, keyed, keyed, sibling("ext#1", "edge#4", math.P3(0, 0, 0), false))

	// The exact edge#3 is gone; one sibling under ext#1 survives (renamed edge#9).
	src.entities = []Entity{sibling("ext#1", "edge#9", math.P3(0, 0, 0), false)}

	got, match := m.BindKeyToObject(key)
	if match != MatchAncestral {
		t.Fatalf("match = %v, want ancestral", match)
	}
	if string(got.Lineage().LineageKey()) != "ext#1/edge#9" {
		t.Errorf("bound to %q, want ext#1/edge#9", got.Lineage().LineageKey())
	}
}

func TestAncestralMissWhenNoSiblingSharesParent(t *testing.T) {
	keyed := sibling("ext#1", "edge#3", math.P3(0, 0, 0), false)
	m, src, key := mintFor(t, keyed, keyed)

	// Survivor belongs to a different parent — not a sibling, so the reference is lost.
	src.entities = []Entity{sibling("ext#2", "edge#3", math.P3(0, 0, 0), false)}

	if _, match := m.BindKeyToObject(key); match != MatchNone {
		t.Fatalf("match = %v, want none", match)
	}
}

func TestRootLineageHasNoAncestralFallback(t *testing.T) {
	keyed := kinNode{kind: KindFace, lin: tokenLineage{key: "base", parent: ""}}
	m, src, key := mintFor(t, keyed, keyed)

	src.entities = []Entity{kinNode{kind: KindFace, lin: tokenLineage{key: "base2", parent: ""}}}

	if _, match := m.BindKeyToObject(key); match != MatchNone {
		t.Fatalf("match = %v, want none (root has no parent to recover via)", match)
	}
}

func TestAncestralSiblingMustMatchKind(t *testing.T) {
	keyed := sibling("ext#1", "edge#3", math.P3(0, 0, 0), false)
	m, src, key := mintFor(t, keyed, keyed)

	// Same parent, but a vertex — not a candidate for an edge key.
	src.entities = []Entity{kinNode{kind: KindVertex, lin: tokenLineage{key: "ext#1/vert#0", parent: "ext#1"}}}

	if _, match := m.BindKeyToObject(key); match != MatchNone {
		t.Fatalf("match = %v, want none (kind must match)", match)
	}
}

// --- geometric tier ---

func TestGeometricPicksNearestAmbiguousSibling(t *testing.T) {
	keyed := sibling("ext#1", "edge#3", math.P3(10, 0, 0), true)
	m, src, key := mintFor(t, keyed, keyed)

	// Exact gone; two siblings survive — disambiguate by nearness to the mint anchor.
	far := sibling("ext#1", "edge#7", math.P3(-10, 0, 0), true)
	near := sibling("ext#1", "edge#8", math.P3(9, 0, 0), true)
	src.entities = []Entity{far, near}

	got, match := m.BindKeyToObject(key)
	if match != MatchGeometric {
		t.Fatalf("match = %v, want geometric", match)
	}
	if string(got.Lineage().LineageKey()) != "ext#1/edge#8" {
		t.Errorf("bound to %q, want the nearer ext#1/edge#8", got.Lineage().LineageKey())
	}
}

func TestGeometricMissOnEquidistantTie(t *testing.T) {
	keyed := sibling("ext#1", "edge#3", math.P3(0, 0, 0), true)
	m, src, key := mintFor(t, keyed, keyed)

	// Two equidistant siblings — no defensible winner.
	src.entities = equidistantSiblings()

	if _, match := m.BindKeyToObject(key); match != MatchNone {
		t.Fatalf("match = %v, want none (geometric tie is not defensible)", match)
	}
}

func TestGeometricMissWithoutAnchor(t *testing.T) {
	keyed := sibling("ext#1", "edge#3", math.P3(0, 0, 0), false) // no anchor captured
	m, src, key := mintFor(t, keyed, keyed)

	src.entities = equidistantSiblings()

	if _, match := m.BindKeyToObject(key); match != MatchNone {
		t.Fatalf("match = %v, want none (no anchor to disambiguate)", match)
	}
}

// --- exact path is unchanged by the new tiers ---

func TestExactStillWinsOverSiblings(t *testing.T) {
	keyed := sibling("ext#1", "edge#3", math.P3(0, 0, 0), true)
	m, src, key := mintFor(t, keyed, keyed, sibling("ext#1", "edge#4", math.P3(1, 0, 0), true))

	// edge#3 still present alongside its sibling — exact must win.
	src.entities = []Entity{sibling("ext#1", "edge#4", math.P3(1, 0, 0), true), keyed}
	got, match := m.BindKeyToObject(key)
	if match != MatchExact {
		t.Fatalf("match = %v, want exact", match)
	}
	if string(got.Lineage().LineageKey()) != "ext#1/edge#3" {
		t.Errorf("bound to %q, want ext#1/edge#3", got.Lineage().LineageKey())
	}
}

// --- health policy: fallback is Warning, true loss is Sick ---

func TestResolveWarnsOnFallbackAndSickensOnLoss(t *testing.T) {
	keyed := sibling("ext#1", "edge#3", math.P3(0, 0, 0), false)
	m, src, key := mintFor(t, keyed, keyed)

	// Lone sibling survives → ancestral → Warning, entity returned.
	src.entities = []Entity{sibling("ext#1", "edge#9", math.P3(0, 0, 0), false)}
	got, h := m.Resolve(key)
	if got == nil || h.Status != health.Warning {
		t.Fatalf("Resolve fallback = (%v, %v), want a healed entity with Warning", got, h.Status)
	}

	// Nothing under the parent survives → Sick, nil entity.
	src.entities = []Entity{sibling("ext#2", "edge#9", math.P3(0, 0, 0), false)}
	got, h = m.Resolve(key)
	if got != nil || h.Status != health.Sick {
		t.Fatalf("Resolve loss = (%v, %v), want nil with Sick", got, h.Status)
	}
}

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
