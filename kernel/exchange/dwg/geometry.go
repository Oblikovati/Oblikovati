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

// Ellipse is a (possibly partial) ellipse. MajorAxis is the major-axis endpoint
// relative to Center; AxisRatio is minor/major; angles are in radians.
type Ellipse struct {
	Handle     uint64
	Center     [3]float64
	MajorAxis  [3]float64
	AxisRatio  float64
	StartAngle float64
	EndAngle   float64
	Normal     [3]float64
}

func (e *Ellipse) EntityHandle() uint64   { return e.Handle }
func (e *Ellipse) EntityType() ObjectType { return TypeEllipse }

// LwPolyline is a lightweight polyline: a vertex list with optional per-vertex
// bulges (tan of a quarter of the included arc angle; 0 = straight to the next
// vertex). It is planar at Elevation, oriented by Normal. Closed joins the last
// vertex back to the first.
type LwPolyline struct {
	Handle    uint64
	Closed    bool
	Elevation float64
	Points    [][2]float64
	Bulges    []float64
	Normal    [3]float64
}

func (e *LwPolyline) EntityHandle() uint64   { return e.Handle }
func (e *LwPolyline) EntityType() ObjectType { return TypeLwpolyline }

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
	case TypeEllipse:
		e = decodeEllipse(r, header.Handle)
	case TypeLwpolyline:
		e = decodeLwPolyline(r, header.Handle, version)
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

// decodeEllipse reads ELLIPSE geometry: centre, major-axis vector, extrusion,
// axis ratio, then start and end angles.
func decodeEllipse(r *BitReader, handle uint64) *Ellipse {
	center := r.Read3BD()
	major := r.Read3BD()
	normal := r.Read3BD()
	ratio := r.ReadBD()
	start := r.ReadBD()
	end := r.ReadBD()
	return &Ellipse{Handle: handle, Center: center, MajorAxis: major, AxisRatio: ratio, StartAngle: start, EndAngle: end, Normal: normal}
}

// lwpolyFlag bits gate the optional LWPOLYLINE fields (ODA).
const (
	lwClosed      = 0x200
	lwConstWidth  = 0x004
	lwElevation   = 0x008
	lwThickness   = 0x002
	lwExtrusion   = 0x001
	lwHasBulges   = 0x010
	lwHasVertexID = 0x400
	lwHasWidths   = 0x020
)

// decodeLwPolyline reads LWPOLYLINE geometry. After the flag-gated header fields,
// the vertices are a 2DD vector (first point full, the rest deltas from the
// previous), then a bulge vector, vertex-id vector (R2010+) and width pairs.
func decodeLwPolyline(r *BitReader, handle uint64, version Version) *LwPolyline {
	flag := r.ReadBS()
	p := &LwPolyline{Handle: handle, Closed: flag&lwClosed != 0, Normal: [3]float64{0, 0, 1}}
	if flag&lwConstWidth != 0 {
		r.ReadBD() // const width
	}
	if flag&lwElevation != 0 {
		p.Elevation = r.ReadBD()
	}
	if flag&lwThickness != 0 {
		r.ReadBD() // thickness
	}
	if flag&lwExtrusion != 0 {
		p.Normal = r.Read3BD()
	}
	numPoints := r.ReadBL()
	var numBulges, numVertexIDs, numWidths int
	if flag&lwHasBulges != 0 {
		numBulges = r.ReadBL()
	}
	if version >= R2010 && flag&lwHasVertexID != 0 {
		numVertexIDs = r.ReadBL()
	}
	if flag&lwHasWidths != 0 {
		numWidths = r.ReadBL()
	}
	p.Points = readLwVertices(r, numPoints)
	p.Bulges = make([]float64, numBulges)
	for i := range p.Bulges {
		p.Bulges[i] = r.ReadBD()
	}
	for i := 0; i < numVertexIDs; i++ {
		r.ReadBL() // vertex ids (unused)
	}
	for i := 0; i < numWidths; i++ {
		r.ReadBD() // start width
		r.ReadBD() // end width
	}
	return p
}

// readLwVertices reads the LWPOLYLINE 2DD point vector: the first vertex is a full
// 2RD, each subsequent vertex a delta (DD) from the previous on each axis.
func readLwVertices(r *BitReader, n int) [][2]float64 {
	if n <= 0 {
		return nil
	}
	pts := make([][2]float64, n)
	pts[0] = r.Read2RD()
	for i := 1; i < n; i++ {
		x := r.ReadDD(pts[i-1][0])
		y := r.ReadDD(pts[i-1][1])
		pts[i] = [2]float64{x, y}
	}
	return pts
}
