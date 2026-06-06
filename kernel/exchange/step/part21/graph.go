// SPDX-License-Identifier: GPL-2.0-only

package part21

import "fmt"

// RawEntity is one parsed DATA statement #id = KEYWORD(params). For a complex
// (multi-type) instance #id = (A(..)B(..)) the Components hold each part; simple
// instances leave Components nil and use Keyword/Params directly.
type RawEntity struct {
	ID         int
	Keyword    string
	Params     []Value
	Components []ComplexPart // non-nil only for complex instances
}

// ComplexPart is one KEYWORD(args) component of a complex (combined) instance.
type ComplexPart struct {
	Keyword string
	Params  []Value
}

// EntityGraph is the resolved DATA section: id→RawEntity, in file order. Resolution
// of references is cycle-safe because it is by id lookup (no eager pointer chasing),
// so the inevitable STEP back-references (face↔loop↔edge) never recurse infinitely.
type EntityGraph struct {
	byID  map[int]*RawEntity
	order []int
}

// newEntityGraph starts an empty graph.
func newEntityGraph() *EntityGraph {
	return &EntityGraph{byID: map[int]*RawEntity{}}
}

// add records an entity, rejecting a duplicate id (a malformed file).
func (g *EntityGraph) add(e *RawEntity) error {
	if _, dup := g.byID[e.ID]; dup {
		return fmt.Errorf("part21: duplicate entity id #%d", e.ID)
	}
	g.byID[e.ID] = e
	g.order = append(g.order, e.ID)
	return nil
}

// Lookup returns the entity for id, or an error naming the missing id (a dangling
// reference) — never a nil-pointer panic.
func (g *EntityGraph) Lookup(id int) (*RawEntity, error) {
	e, ok := g.byID[id]
	if !ok {
		return nil, fmt.Errorf("part21: dangling reference to #%d", id)
	}
	return e, nil
}

// IDs returns the entity ids in file order.
func (g *EntityGraph) IDs() []int {
	out := make([]int, len(g.order))
	copy(out, g.order)
	return out
}

// Len returns the number of entities.
func (g *EntityGraph) Len() int { return len(g.order) }

// EntitiesOfType returns every entity whose (simple) keyword matches, in file order.
func (g *EntityGraph) EntitiesOfType(keyword string) []*RawEntity {
	var out []*RawEntity
	for _, id := range g.order {
		if g.byID[id].Keyword == keyword {
			out = append(out, g.byID[id])
		}
	}
	return out
}
