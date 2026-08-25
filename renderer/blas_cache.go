// SPDX-License-Identifier: GPL-2.0-only

package renderer

// BLASBuilder builds or destroys one backend-specific bottom-level acceleration
// structure for a body's triangle mesh — implemented by the hardware backend
// (head/internal/native's RTScene, PBI-333) so [BLASCache]'s per-body dirty-tracking
// decision stays backend-agnostic and CPU-testable (ADR-0014), matching [Intersector]'s
// own seam. The returned handle is opaque to this package.
type BLASBuilder interface {
	BuildBLAS(triangles []Triangle) (handle any, err error)
	DestroyBLAS(handle any)
}

// BLASCache retains one built BLAS per unique body content hash, so a rebuild happens
// only for a body whose triangle content actually changed — never every frame, and
// never for an unrelated body that happens to share a scene with an edited one (PBI-333's
// "editing one body refits/rebuilds only its own BLAS" guarantee). The hash itself is the
// caller's concern (e.g. a content hash of the tessellated triangle streams, mirroring
// head/ui's existing overlayHash technique) — this cache only decides build/keep/destroy
// from whatever hashes it's handed.
type BLASCache struct {
	builder BLASBuilder
	entries map[uint64]any

	// BuildCount / DestroyCount instrument every Sync call for the PBI-333 rebuild-count
	// regression test — production callers may ignore them.
	BuildCount, DestroyCount int
}

// NewBLASCache returns an empty cache that builds BLASes via builder.
func NewBLASCache(builder BLASBuilder) *BLASCache {
	return &BLASCache{builder: builder, entries: map[uint64]any{}}
}

// Sync ensures exactly the given bodies have a resident BLAS: a hash already cached is
// left untouched (no rebuild); a new or changed hash is built; a previously-cached hash
// no longer present is destroyed (freeing GPU memory for a removed or edited-away body).
// Returns the current hash→handle mapping, e.g. for the caller's TLAS instance build.
func (c *BLASCache) Sync(bodies map[uint64][]Triangle) map[uint64]any {
	for hash := range c.entries {
		if _, present := bodies[hash]; !present {
			c.builder.DestroyBLAS(c.entries[hash])
			delete(c.entries, hash)
			c.DestroyCount++
		}
	}
	for hash, triangles := range bodies {
		if _, cached := c.entries[hash]; cached {
			continue
		}
		handle, err := c.builder.BuildBLAS(triangles)
		if err != nil {
			continue
		}
		c.entries[hash] = handle
		c.BuildCount++
	}
	out := make(map[uint64]any, len(c.entries))
	for hash, handle := range c.entries {
		out[hash] = handle
	}
	return out
}

// Len reports how many BLASes are currently resident.
func (c *BLASCache) Len() int { return len(c.entries) }
