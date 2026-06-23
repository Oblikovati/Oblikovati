// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"

	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// Renaming the other browser-history entities — sketches, 3D sketches, and the user work
// planes/axes/points — mirrors RenameFeature (feature_lifecycle.go): a non-empty name that is
// unique among its siblings, recorded as one undo edit, with the stable id untouched. The
// grounded origin coordinate-system datums (the Origin folder) are not renameable (#1264).

// RenameSketch renames a 2D sketch; the name must be non-empty and unique among the part's 2D
// sketches.
func (s *Session) RenameSketch(sk *sketch.Sketch, name string) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if name == "" {
		return errors.New("app: RenameSketch: name must be non-empty")
	}
	cs := part.Sketches()
	for i := 0; i < cs.Count(); i++ {
		if o := cs.Item(i); o != sk && o.Name() == name {
			return fmt.Errorf("app: RenameSketch: name %q is already used by another sketch", name)
		}
	}
	sk.SetName(name)
	s.recordEdit(part, "Rename Sketch")
	return nil
}

// RenameSketch3D renames a 3D sketch; unique among the part's 3D sketches.
func (s *Session) RenameSketch3D(sk *sketch.Sketch3D, name string) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if name == "" {
		return errors.New("app: RenameSketch3D: name must be non-empty")
	}
	cs := part.Sketches3D()
	for i := 0; i < cs.Count(); i++ {
		if o := cs.Item(i); o != sk && o.Name() == name {
			return fmt.Errorf("app: RenameSketch3D: name %q is already used by another 3D sketch", name)
		}
	}
	sk.SetName(name)
	s.recordEdit(part, "Rename Sketch")
	return nil
}

// RenameWorkPlane renames a user work plane; unique among the part's work planes. An origin
// coordinate-system plane (XY/XZ/YZ) cannot be renamed.
func (s *Session) RenameWorkPlane(wp *feature.WorkPlane, name string) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if wp.IsCoordinateSystemElement() {
		return errors.New("app: RenameWorkPlane: origin coordinate-system planes cannot be renamed")
	}
	if name == "" {
		return errors.New("app: RenameWorkPlane: name must be non-empty")
	}
	cs := part.WorkPlanes()
	for i := 0; i < cs.Count(); i++ {
		if o := cs.Item(i); o != wp && o.Name() == name {
			return fmt.Errorf("app: RenameWorkPlane: name %q is already used by another work plane", name)
		}
	}
	wp.SetName(name)
	s.recordEdit(part, "Rename Work Plane")
	return nil
}

// RenameWorkAxis renames a user work axis; unique among the part's work axes. An origin
// coordinate-system axis (X/Y/Z) cannot be renamed.
func (s *Session) RenameWorkAxis(wa *feature.WorkAxis, name string) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if wa.IsCoordinateSystemElement() {
		return errors.New("app: RenameWorkAxis: origin coordinate-system axes cannot be renamed")
	}
	if name == "" {
		return errors.New("app: RenameWorkAxis: name must be non-empty")
	}
	cs := part.WorkAxes()
	for i := 0; i < cs.Count(); i++ {
		if o := cs.Item(i); o != wa && o.Name() == name {
			return fmt.Errorf("app: RenameWorkAxis: name %q is already used by another work axis", name)
		}
	}
	wa.SetName(name)
	s.recordEdit(part, "Rename Work Axis")
	return nil
}

// RenameWorkPoint renames a user work point; unique among the part's work points. The origin
// centre point cannot be renamed.
func (s *Session) RenameWorkPoint(wp *feature.WorkPoint, name string) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if wp.IsCoordinateSystemElement() {
		return errors.New("app: RenameWorkPoint: the origin centre point cannot be renamed")
	}
	if name == "" {
		return errors.New("app: RenameWorkPoint: name must be non-empty")
	}
	cs := part.WorkPoints()
	for i := 0; i < cs.Count(); i++ {
		if o := cs.Item(i); o != wp && o.Name() == name {
			return fmt.Errorf("app: RenameWorkPoint: name %q is already used by another work point", name)
		}
	}
	wp.SetName(name)
	s.recordEdit(part, "Rename Work Point")
	return nil
}
