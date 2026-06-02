// SPDX-License-Identifier: GPL-2.0-only

package topo

// SurfaceBodies is the collection of bodies a component definition owns (M07-F04).
type SurfaceBodies struct {
	items []*Body
	byID  map[uint64]*Body
}

// NewSurfaceBodies returns an empty body collection.
func NewSurfaceBodies() *SurfaceBodies {
	return &SurfaceBodies{byID: map[uint64]*Body{}}
}

// Add registers a body and returns it.
func (c *SurfaceBodies) Add(b *Body) *Body {
	c.items = append(c.items, b)
	c.byID[b.id] = b
	return b
}

// Count returns the number of bodies; Item returns the i-th.
func (c *SurfaceBodies) Count() int       { return len(c.items) }
func (c *SurfaceBodies) Item(i int) *Body { return c.items[i] }

// ByID returns the body with the given session id.
func (c *SurfaceBodies) ByID(id uint64) (*Body, bool) {
	b, ok := c.byID[id]
	return b, ok
}

// All returns a snapshot of the bodies.
func (c *SurfaceBodies) All() []*Body {
	out := make([]*Body, len(c.items))
	copy(out, c.items)
	return out
}

// Remove deletes a body, reporting whether it was present.
func (c *SurfaceBodies) Remove(b *Body) bool {
	if _, ok := c.byID[b.id]; !ok {
		return false
	}
	delete(c.byID, b.id)
	for i, existing := range c.items {
		if existing == b {
			c.items = append(c.items[:i], c.items[i+1:]...)
			return true
		}
	}
	return false
}
