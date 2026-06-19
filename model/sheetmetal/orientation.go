// SPDX-License-Identifier: GPL-2.0-only

package sheetmetal

import (
	"fmt"

	"oblikovati.org/api/types"
)

// Flat-pattern orientations (M13-F05, #635). An orientation is a saved alignment state of the
// developed flat: an alignment axis (a topology reference key, or none for the part's natural
// axes) laid horizontal or vertical, an extra alignment rotation, and flips for the alignment
// direction and the base face. The active orientation frames the flat for drawing views and
// export and drives its reported length/width. Orientations are container/value objects, not
// history features (they have no Definition→Add→Feature triangle).

// AlignmentType is the canonical Apache-2.0 enum (ADR-0018), aliased so call sites read
// sheetmetal.HorizontalAlignment.
type AlignmentType = types.AlignmentType

const (
	HorizontalAlignment = types.HorizontalAlignment
	VerticalAlignment   = types.VerticalAlignment
)

// DefaultOrientationName is the seeded orientation every sheet-metal part starts with; it
// cannot be deleted.
const DefaultOrientationName = "Flat Pattern Default"

// FlatPatternOrientation is one named alignment state.
type FlatPatternOrientation struct {
	Name              string
	AlignmentType     AlignmentType
	AlignmentRotation float64 // extra rotation applied after the axis is aligned (radians)
	AlignmentAxisKey  []byte  // topology reference key of the alignment edge/axis; nil ⇒ natural axes
	FlipAlignmentAxis bool
	FlipBaseFace      bool
}

// Orientations is the ordered set of a part's flat-pattern orientations with one active.
type Orientations struct {
	items  []*FlatPatternOrientation
	active int
}

// NewOrientations returns a set seeded with the (active) default orientation.
func NewOrientations() *Orientations {
	return &Orientations{items: []*FlatPatternOrientation{{Name: DefaultOrientationName}}}
}

// List returns the orientations in creation order.
func (o *Orientations) List() []*FlatPatternOrientation { return o.items }

// Active returns the active orientation (never nil for a seeded set).
func (o *Orientations) Active() *FlatPatternOrientation { return o.items[o.active] }

// ByName returns the named orientation and its index, or ok=false.
func (o *Orientations) ByName(name string) (*FlatPatternOrientation, int, bool) {
	for i, it := range o.items {
		if it.Name == name {
			return it, i, true
		}
	}
	return nil, 0, false
}

// Add appends a new orientation, erroring on a blank or duplicate name. It does not activate.
func (o *Orientations) Add(or *FlatPatternOrientation) error {
	if or.Name == "" {
		return fmt.Errorf("flat-pattern orientation: name is required")
	}
	if _, _, ok := o.ByName(or.Name); ok {
		return fmt.Errorf("flat-pattern orientation %q already exists", or.Name)
	}
	o.items = append(o.items, or)
	return nil
}

// Activate makes the named orientation current, erroring if it is unknown.
func (o *Orientations) Activate(name string) error {
	if _, i, ok := o.ByName(name); ok {
		o.active = i
		return nil
	}
	return fmt.Errorf(errNoFlatOrientation, name)
}

// Copy duplicates the named orientation under newName (defaulting to "<name> Copy"), erroring
// on an unknown source or a duplicate target.
func (o *Orientations) Copy(name, newName string) (*FlatPatternOrientation, error) {
	src, _, ok := o.ByName(name)
	if !ok {
		return nil, fmt.Errorf(errNoFlatOrientation, name)
	}
	if newName == "" {
		newName = name + " Copy"
	}
	dup := *src
	dup.Name = newName
	if err := o.Add(&dup); err != nil {
		return nil, err
	}
	return &dup, nil
}

// Delete removes the named orientation; the default orientation cannot be deleted. Deleting
// the active orientation falls back to the default.
func (o *Orientations) Delete(name string) error {
	if name == DefaultOrientationName {
		return fmt.Errorf("the default flat-pattern orientation cannot be deleted")
	}
	_, i, ok := o.ByName(name)
	if !ok {
		return fmt.Errorf(errNoFlatOrientation, name)
	}
	o.items = append(o.items[:i], o.items[i+1:]...)
	if o.active == i {
		o.active = 0
	} else if o.active > i {
		o.active--
	}
	return nil
}

// IsActive reports whether or is the active orientation.
func (o *Orientations) IsActive(or *FlatPatternOrientation) bool { return o.Active() == or }
