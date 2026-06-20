// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"fmt"

	"oblikovati.org/math"
)

// Crops limit which of a cloud's points display: a model-space box volume the user draws to focus
// on a region of the scan (M17-F06, #645). When a cloud has one or more ACTIVE crops, only points
// inside at least one active crop render (the crops union); with no active crop, every point shows.
// Inactive crops are remembered but do not filter, so a crop can be toggled without redefining it.

// PointCloudCrop is one named crop volume — a model-space axis-aligned box plus an active flag.
type PointCloudCrop struct {
	name   string
	box    math.Box
	active bool
}

// Name/Box/Active report the crop's identity, model-space bounds, and whether it filters display.
func (c *PointCloudCrop) Name() string  { return c.name }
func (c *PointCloudCrop) Box() math.Box { return c.box }
func (c *PointCloudCrop) Active() bool  { return c.active }

// SetActive toggles whether the crop limits display; SetBox redefines its volume.
func (c *PointCloudCrop) SetActive(a bool)  { c.active = a }
func (c *PointCloudCrop) SetBox(b math.Box) { c.box = b }

// PointCloudCrops is a cloud's crop-volume collection.
type PointCloudCrops struct {
	items []*PointCloudCrop
}

// Count returns the number of crops; Item returns the i-th in creation order.
func (cs *PointCloudCrops) Count() int                 { return len(cs.items) }
func (cs *PointCloudCrops) Item(i int) *PointCloudCrop { return cs.items[i] }

// Names returns the crop names in creation order.
func (cs *PointCloudCrops) Names() []string {
	out := make([]string, len(cs.items))
	for i, c := range cs.items {
		out[i] = c.name
	}
	return out
}

// ByName returns the named crop, ok=false when absent.
func (cs *PointCloudCrops) ByName(name string) (*PointCloudCrop, bool) {
	for _, c := range cs.items {
		if c.name == name {
			return c, true
		}
	}
	return nil, false
}

// Add appends an active crop over box under name; an empty name is rejected, a duplicate errors
// via the returned ok=false. The crop starts active so a freshly drawn crop immediately focuses.
func (cs *PointCloudCrops) Add(name string, box math.Box) (*PointCloudCrop, bool) {
	if name == "" {
		return nil, false
	}
	if _, exists := cs.ByName(name); exists {
		return nil, false
	}
	c := &PointCloudCrop{name: name, box: box, active: true}
	cs.items = append(cs.items, c)
	return c, true
}

// Remove deletes the named crop, reporting whether it existed.
func (cs *PointCloudCrops) Remove(name string) bool {
	for i, c := range cs.items {
		if c.name == name {
			cs.items = append(cs.items[:i], cs.items[i+1:]...)
			return true
		}
	}
	return false
}

// anyActive reports whether at least one crop is currently filtering display.
func (cs *PointCloudCrops) anyActive() bool {
	for _, c := range cs.items {
		if c.active {
			return true
		}
	}
	return false
}

// Admits reports whether a model-space point passes the active crops: true when no crop is active
// (unfiltered), else true only inside at least one active crop.
func (cs *PointCloudCrops) Admits(p math.Point3) bool {
	anyActive := false
	for _, c := range cs.items {
		if !c.active {
			continue
		}
		anyActive = true
		if c.box.Contains(p) {
			return true
		}
	}
	return !anyActive
}

// uniqueName returns base suffixed with the smallest positive integer free in the collection.
func (cs *PointCloudCrops) uniqueName(base string) string {
	for i := 1; ; i++ {
		name := fmt.Sprintf("%s%d", base, i)
		if _, exists := cs.ByName(name); !exists {
			return name
		}
	}
}
