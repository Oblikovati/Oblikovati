// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "fmt"

// Entity is a decoded DWG drawing entity. The intermediate model is deliberately
// geometry-only (no DWG styling/handles beyond identity): it is the bridge the
// Sketch/Sketch3D converter consumes, so it carries just what sketch geometry
// needs. Coordinates are in the drawing's units (model space).
type Entity interface {
	// Handle is the object's DWG handle (its identity in the file).
	EntityHandle() uint64
	// EntityType is the DWG type code.
	EntityType() ObjectType
}

// Line is a straight segment between two 3D points.
type Line struct {
	Handle uint64
	Start  [3]float64
	End    [3]float64
}

func (e *Line) EntityHandle() uint64   { return e.Handle }
func (e *Line) EntityType() ObjectType { return TypeLine }

// Circle is a full circle: a centre, radius and normal (extrusion) direction.
type Circle struct {
	Handle uint64
	Center [3]float64
	Radius float64
	Normal [3]float64
}

func (e *Circle) EntityHandle() uint64   { return e.Handle }
func (e *Circle) EntityType() ObjectType { return TypeCircle }

// Arc is a circular arc, angles in radians measured in the plane of Normal.
type Arc struct {
	Handle     uint64
	Center     [3]float64
	Radius     float64
	StartAngle float64
	EndAngle   float64
	Normal     [3]float64
}

func (e *Arc) EntityHandle() uint64   { return e.Handle }
func (e *Arc) EntityType() ObjectType { return TypeArc }

// Point is a single model-space point.
type Point struct {
	Handle   uint64
	Position [3]float64
}

func (e *Point) EntityHandle() uint64   { return e.Handle }
func (e *Point) EntityType() ObjectType { return TypePoint }

// decodeEntity decodes one geometry entity's coordinates from a reader positioned
// at its type-specific data (see seekEntityGeometry). It returns (nil, nil) for a
// type whose geometry decoder is not yet implemented, so callers can skip it.
func decodeEntity(r *BitReader, header ObjectHeader, version Version) (Entity, error) {
	var e Entity
	switch header.Type {
	case TypeLine:
		e = decodeLine(r, header.Handle, version)
	case TypeCircle:
		e = decodeCircle(r, header.Handle)
	case TypeArc:
		e = decodeArc(r, header.Handle)
	case TypePoint:
		e = decodePoint(r, header.Handle)
	default:
		return nil, nil
	}
	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("dwg: %s handle %d geometry: %w", header.Type.Name(), header.Handle, err)
	}
	return e, nil
}

// decodeLine reads LINE geometry (ODA): a z-is-zero flag, then start.x/start.y as
// raw doubles with end.x/end.y as deltas from them, optional z, then thickness and
// extrusion (consumed but unused here).
func decodeLine(r *BitReader, handle uint64, version Version) *Line {
	zZero := r.ReadBit() == 1
	sx := r.ReadRD()
	ex := r.ReadDD(sx)
	sy := r.ReadRD()
	ey := r.ReadDD(sy)
	var sz, ez float64
	if !zZero {
		sz = r.ReadRD()
		ez = r.ReadDD(sz)
	}
	r.ReadBT() // thickness
	r.ReadBE() // extrusion
	return &Line{Handle: handle, Start: [3]float64{sx, sy, sz}, End: [3]float64{ex, ey, ez}}
}

// decodeCircle reads CIRCLE geometry: centre, radius, thickness, extrusion.
func decodeCircle(r *BitReader, handle uint64) *Circle {
	center := r.Read3BD()
	radius := r.ReadBD()
	r.ReadBT() // thickness
	normal := r.ReadBE()
	return &Circle{Handle: handle, Center: center, Radius: radius, Normal: normal}
}

// decodeArc reads ARC geometry: centre, radius, thickness, extrusion, then start
// and end angles.
func decodeArc(r *BitReader, handle uint64) *Arc {
	center := r.Read3BD()
	radius := r.ReadBD()
	r.ReadBT() // thickness
	normal := r.ReadBE()
	start := r.ReadBD()
	end := r.ReadBD()
	return &Arc{Handle: handle, Center: center, Radius: radius, StartAngle: start, EndAngle: end, Normal: normal}
}

// decodePoint reads POINT geometry: position, thickness, extrusion, x-angle.
func decodePoint(r *BitReader, handle uint64) *Point {
	x := r.ReadBD()
	y := r.ReadBD()
	z := r.ReadBD()
	r.ReadBT() // thickness
	r.ReadBE() // extrusion
	r.ReadBD() // x-angle
	return &Point{Handle: handle, Position: [3]float64{x, y, z}}
}
