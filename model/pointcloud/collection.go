// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"fmt"

	"oblikovati.org/math"
)

// PointClouds is a component definition's collection of attached scans (M17-F06, #645). Names are
// unique within a definition so other objects (and the browser) bind to a stable handle.
type PointClouds struct {
	items []*PointCloud
}

// NewPointClouds creates an empty collection.
func NewPointClouds() *PointClouds { return &PointClouds{} }

// Count returns the number of attached clouds; Item returns the i-th in attach order.
func (c *PointClouds) Count() int             { return len(c.items) }
func (c *PointClouds) Item(i int) *PointCloud { return c.items[i] }

// Names returns the cloud names in attach order.
func (c *PointClouds) Names() []string {
	out := make([]string, len(c.items))
	for i, pc := range c.items {
		out[i] = pc.name
	}
	return out
}

// ByName returns the named cloud, ok=false when absent.
func (c *PointClouds) ByName(name string) (*PointCloud, bool) {
	for _, pc := range c.items {
		if pc.name == name {
			return pc, true
		}
	}
	return nil, false
}

// Add attaches a cloud built from decoded cloud-local points under a unique name, erroring on a
// duplicate name. source/resourceID record the scan's origin path and its embedded-bytes id.
func (c *PointClouds) Add(name, source, resourceID string, points []math.Point3) (*PointCloud, error) {
	return c.AddWithSamples(name, source, resourceID, pointSamples(points))
}

// AddWithSamples attaches a cloud built from decoded scan samples under a unique name, erroring
// on a duplicate name. source/resourceID record the scan's origin path and its embedded-bytes id.
func (c *PointClouds) AddWithSamples(name, source, resourceID string, samples []PointSample) (*PointCloud, error) {
	if name == "" {
		return nil, fmt.Errorf("pointcloud: a cloud needs a non-empty name")
	}
	if _, exists := c.ByName(name); exists {
		return nil, fmt.Errorf("pointcloud: a cloud named %q already exists", name)
	}
	pc := NewWithSamples(name, source, resourceID, samples)
	c.items = append(c.items, pc)
	return pc, nil
}

func pointSamples(points []math.Point3) []PointSample {
	samples := make([]PointSample, len(points))
	for i, p := range points {
		samples[i] = PointSample{Point: p}
	}
	return samples
}

// Append re-attaches an already-built cloud (used by persistence restore, which reconstructs the
// cloud from the recipe). It enforces the same unique-name rule as Add.
func (c *PointClouds) Append(pc *PointCloud) error {
	if pc == nil {
		return fmt.Errorf("pointcloud: Append(nil)")
	}
	if _, exists := c.ByName(pc.name); exists {
		return fmt.Errorf("pointcloud: a cloud named %q already exists", pc.name)
	}
	c.items = append(c.items, pc)
	return nil
}

// Remove deletes the named cloud, reporting whether it existed.
func (c *PointClouds) Remove(name string) bool {
	for i, pc := range c.items {
		if pc.name == name {
			c.items = append(c.items[:i], c.items[i+1:]...)
			return true
		}
	}
	return false
}

// UniqueName returns base suffixed with the smallest positive integer that is not already a cloud
// name (Cloud1, Cloud2, …), so an attach with a colliding base still gets a distinct name.
func (c *PointClouds) UniqueName(base string) string {
	for i := 1; ; i++ {
		name := fmt.Sprintf("%s%d", base, i)
		if _, exists := c.ByName(name); !exists {
			return name
		}
	}
}
