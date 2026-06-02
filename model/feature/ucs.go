// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// UserCoordinateSystem is a local modeling frame — a triad of origin, X/Y/Z axes
// (and the implied XY/XZ/YZ planes) features and constraints can work relative to.
type UserCoordinateSystem struct {
	id      ID
	name    string
	origin  math.Point3
	x, y, z math.UnitVector3
}

func (u *UserCoordinateSystem) ID() ID           { return u.id }
func (u *UserCoordinateSystem) Name() string     { return u.name }
func (u *UserCoordinateSystem) SetName(n string) { u.name = n }

// Origin and the axes define the frame.
func (u *UserCoordinateSystem) Origin() math.Point3     { return u.origin }
func (u *UserCoordinateSystem) XAxis() math.UnitVector3 { return u.x }
func (u *UserCoordinateSystem) YAxis() math.UnitVector3 { return u.y }
func (u *UserCoordinateSystem) ZAxis() math.UnitVector3 { return u.z }

// XYPlane returns the frame's XY plane, usable as a sketch plane.
func (u *UserCoordinateSystem) XYPlane() sketch.Plane {
	p, _ := sketch.NewPlane(u.origin, u.x, u.y) // axes are orthonormal by construction
	return p
}

// UserCoordinateSystems is the collection of local frames.
type UserCoordinateSystems struct {
	items []*UserCoordinateSystem
	byID  map[ID]*UserCoordinateSystem
}

// NewUserCoordinateSystems returns an empty collection.
func NewUserCoordinateSystems() *UserCoordinateSystems {
	return &UserCoordinateSystems{byID: map[ID]*UserCoordinateSystem{}}
}

// AddByPlane creates a UCS aligned with a plane: origin and X/Y from the plane, Z
// along its normal.
func (c *UserCoordinateSystems) AddByPlane(plane sketch.Plane) *UserCoordinateSystem {
	u := &UserCoordinateSystem{
		id: nextID(), name: "UCS",
		origin: plane.Origin(), x: plane.XAxis(), y: plane.YAxis(), z: plane.Normal(),
	}
	c.items = append(c.items, u)
	c.byID[u.id] = u
	return u
}

// Count/Item index the collection.
func (c *UserCoordinateSystems) Count() int                       { return len(c.items) }
func (c *UserCoordinateSystems) Item(i int) *UserCoordinateSystem { return c.items[i] }
