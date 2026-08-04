// SPDX-License-Identifier: GPL-2.0-only

// Package drawing is the format-neutral 2D/3D drawing model shared by the DWG and DXF
// codecs. It carries only geometry — the curve entities a CAD drawing exchanges and the
// metadata (units) needed to bring them into a document — so a decoder (dwg or dxf) can
// populate one model and the Sketch/Sketch3D converter can consume it without knowing the
// source format. Coordinates are in the drawing's own units (model space); see
// MetersPerUnit for the $INSUNITS conversion.
package drawing

// Entity is one drawing entity in the neutral model. It is deliberately geometry-only
// (no styling, no format-specific type codes): the bridge both codecs and the sketch
// converter share. Identify a concrete entity with a Go type switch, or with Kind for a
// stable, format-neutral discriminator/name.
type Entity interface {
	// EntityHandle is the entity's identity in its source file (DWG handle / DXF code-5
	// hex handle). 0 when the source carried none.
	EntityHandle() uint64
	// Kind is the format-neutral entity discriminator (its canonical name via String).
	Kind() Kind
}

// Kind is a format-neutral entity discriminator. Its String is the canonical entity name
// (LINE, CIRCLE, …) — which doubles as the DXF entity name, so the DXF codec reuses it.
type Kind int

const (
	KindUnknown Kind = iota
	KindLine
	KindCircle
	KindArc
	KindPoint
	KindEllipse
	KindLwPolyline
	KindSpline
	KindInsert
	KindText
)

// kindNames maps each Kind to its canonical (DXF) entity name.
var kindNames = map[Kind]string{
	KindUnknown: "UNKNOWN", KindLine: "LINE", KindCircle: "CIRCLE", KindArc: "ARC",
	KindPoint: "POINT", KindEllipse: "ELLIPSE", KindLwPolyline: "LWPOLYLINE",
	KindSpline: "SPLINE", KindInsert: "INSERT", KindText: "TEXT",
}

// String returns the canonical entity name, or "UNKNOWN" for an unrecognised kind.
func (k Kind) String() string {
	if name, ok := kindNames[k]; ok {
		return name
	}
	return "UNKNOWN"
}

// Drawing is the decoded, sketch-relevant content of a drawing file: its model-space curve
// entities and the unit code that scales them. It is the hand-off to the Sketch/Sketch3D
// converter.
type Drawing struct {
	Entities []Entity
	// Units is the $INSUNITS code (0 = unitless); use MetersPerUnit to convert.
	Units int
	// Layers is the drawing's layer table — the formatting entities inherit (#2015).
	Layers []Layer
	// Styles is each entity's explicit formatting, keyed by entity handle. Absent means the
	// entity inherits everything from its layer, which is the common case; see [Style].
	Styles map[uint64]Style
	// EntityLayers is which layer each entity sits on, keyed by entity handle.
	EntityLayers map[uint64]string
}

// Line is a straight segment between two 3D points.
type Line struct {
	Handle uint64
	Layer  string // target layer name ("" ⇒ the default "0" layer)
	Start  [3]float64
	End    [3]float64
}

func (e *Line) EntityHandle() uint64 { return e.Handle }
func (e *Line) Kind() Kind           { return KindLine }

// Circle is a full circle: a centre, radius and normal (extrusion) direction.
type Circle struct {
	Handle uint64
	Layer  string
	Center [3]float64
	Radius float64
	Normal [3]float64
}

func (e *Circle) EntityHandle() uint64 { return e.Handle }
func (e *Circle) Kind() Kind           { return KindCircle }

// Arc is a circular arc, angles in radians measured in the plane of Normal.
type Arc struct {
	Handle     uint64
	Layer      string
	Center     [3]float64
	Radius     float64
	StartAngle float64
	EndAngle   float64
	Normal     [3]float64
}

func (e *Arc) EntityHandle() uint64 { return e.Handle }
func (e *Arc) Kind() Kind           { return KindArc }

// Point is a single model-space point.
type Point struct {
	Handle   uint64
	Layer    string
	Position [3]float64
}

func (e *Point) EntityHandle() uint64 { return e.Handle }
func (e *Point) Kind() Kind           { return KindPoint }

// Ellipse is a (possibly partial) ellipse. MajorAxis is the major-axis endpoint
// relative to Center; AxisRatio is minor/major; angles are parametric, in radians.
type Ellipse struct {
	Handle     uint64
	Layer      string
	Center     [3]float64
	MajorAxis  [3]float64
	AxisRatio  float64
	StartAngle float64
	EndAngle   float64
	Normal     [3]float64
}

func (e *Ellipse) EntityHandle() uint64 { return e.Handle }
func (e *Ellipse) Kind() Kind           { return KindEllipse }

// LwPolyline is a lightweight polyline: a vertex list with optional per-vertex bulges
// (tan of a quarter of the included arc angle; 0 = straight to the next vertex). It is
// planar at Elevation, oriented by Normal. Closed joins the last vertex back to the first.
type LwPolyline struct {
	Handle    uint64
	Layer     string
	Closed    bool
	Elevation float64
	Points    [][2]float64
	Bulges    []float64
	Normal    [3]float64
}

func (e *LwPolyline) EntityHandle() uint64 { return e.Handle }
func (e *LwPolyline) Kind() Kind           { return KindLwPolyline }

// Spline is a NURBS curve. Scenario 1 stores control points (with optional weights) and a
// knot vector; scenario 2 stores fit points (the converter rebuilds the curve from
// whichever is present).
type Spline struct {
	Handle        uint64
	Layer         string
	Degree        int
	Closed        bool
	Rational      bool
	Knots         []float64
	ControlPoints [][3]float64
	Weights       []float64 // empty when not rational/weighted
	FitPoints     [][3]float64
}

func (e *Spline) EntityHandle() uint64 { return e.Handle }
func (e *Spline) Kind() Kind           { return KindSpline }

// Insert is a block reference: it places the entities of the block identified by
// BlockHeader at Insertion, after applying Scale and a Rotation (radians, about the
// extrusion/Z axis). It is an intermediate — a codec expands it into transformed copies of
// the block's geometry, so an Insert never reaches the sketch converter. (For DXF, where
// blocks are keyed by name, the codec assigns a synthetic handle per block name into
// BlockHeader so this struct stays format-neutral.)
type Insert struct {
	Handle      uint64
	BlockHeader uint64 // handle of the referenced block's definition
	Insertion   [3]float64
	Scale       [3]float64
	Rotation    float64
}

func (e *Insert) EntityHandle() uint64 { return e.Handle }
func (e *Insert) Kind() Kind           { return KindInsert }

// Text is a single-line annotation string placed at Position, of the given Height (drawing
// units) and Rotation (radians, about Z). It carries no geometry of its own — exporters use
// it to tag drawings (e.g. a flat pattern's punch tokens). Importers that only rebuild curve
// geometry skip it.
type Text struct {
	Handle   uint64
	Layer    string
	Position [3]float64
	Height   float64
	Value    string
	Rotation float64
}

func (e *Text) EntityHandle() uint64 { return e.Handle }
func (e *Text) Kind() Kind           { return KindText }
