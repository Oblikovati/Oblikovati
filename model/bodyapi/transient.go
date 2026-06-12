// SPDX-License-Identifier: GPL-2.0-only

package bodyapi

import (
	"fmt"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

var (
	_ contract.TransientBRep = (*TransientBRep)(nil)
	_ contract.TransientBody = (*TransientBody)(nil)
)

// TransientBody is one ownerless body in the session registry.
type TransientBody struct {
	handle int
	body   *topo.Body
	q      ops.Quality
}

// Handle returns the session handle; Topo the kernel body.
func (t *TransientBody) Handle() int      { return t.handle }
func (t *TransientBody) Topo() *topo.Body { return t.body }

func (t *TransientBody) IsSolid() bool    { return t.body.IsSolid() }
func (t *TransientBody) FaceCount() int   { return len(t.body.Faces()) }
func (t *TransientBody) EdgeCount() int   { return len(t.body.Edges()) }
func (t *TransientBody) VertexCount() int { return len(t.body.Vertices()) }
func (t *TransientBody) ShellCount() int  { return len(t.body.Shells()) }
func (t *TransientBody) WireCount() int   { return len(t.body.Wires()) }

// Volume is the enclosed volume in cm³ (0 for an open surface body).
func (t *TransientBody) Volume() float64 {
	if !t.body.IsSolid() {
		return 0
	}
	return ops.BodyGeometryProperties(t.body, t.q).Volume
}

// TransientBRep is the session's transient body factory and registry
// (M07-F05, #628). Handles are positive and never reused within a session.
type TransientBRep struct {
	next   int
	bodies map[int]*TransientBody
	q      ops.Quality
}

// NewTransientBRep returns an empty registry at the given quality.
//
// Example: tb := bodyapi.NewTransientBRep(ops.DefaultQuality())
func NewTransientBRep(q ops.Quality) *TransientBRep {
	return &TransientBRep{next: 1, bodies: map[int]*TransientBody{}, q: q}
}

// register adopts a kernel body under a fresh handle.
func (t *TransientBRep) register(b *topo.Body) *TransientBody {
	tb := &TransientBody{handle: t.next, body: b, q: t.q}
	t.bodies[t.next] = tb
	t.next++
	return tb
}

// Adopt registers an externally built kernel body (the wire layer's copies of
// document bodies, offset results, sections).
func (t *TransientBRep) Adopt(b *topo.Body) *TransientBody { return t.register(b) }

// ByHandle resolves a handle.
func (t *TransientBRep) ByHandle(h int) (*TransientBody, bool) {
	tb, ok := t.bodies[h]
	return tb, ok
}

// Handles lists the live handles, ascending.
func (t *TransientBRep) Handles() []int {
	out := make([]int, 0, len(t.bodies))
	for h := 1; h < t.next; h++ {
		if _, ok := t.bodies[h]; ok {
			out = append(out, h)
		}
	}
	return out
}

// Delete frees a handle; reports whether it existed.
func (t *TransientBRep) Delete(h int) bool {
	_, ok := t.bodies[h]
	delete(t.bodies, h)
	return ok
}

// Replace swaps a handle's body in place (the DoBoolean blank semantics).
func (t *TransientBRep) Replace(h int, b *topo.Body) error {
	tb, ok := t.bodies[h]
	if !ok {
		return fmt.Errorf("bodyapi: no transient body with handle %d", h)
	}
	tb.body = b
	return nil
}

// CreateSolidBlock builds the box between two opposite corners.
func (t *TransientBRep) CreateSolidBlock(min, max types.Point) (contract.TransientBody, error) {
	b, err := brep.SolidBlock(p3(min), p3(max), "transient")
	if err != nil {
		return nil, err
	}
	return t.register(b), nil
}

// CreateSolidCylinderCone builds a cylinder/cone between two section centers.
func (t *TransientBRep) CreateSolidCylinderCone(bottom, top types.Point, bottomRadius, topRadius float64) (contract.TransientBody, error) {
	b, err := brep.SolidCylinderCone(p3(bottom), p3(top), bottomRadius, topRadius, "transient")
	if err != nil {
		return nil, err
	}
	return t.register(b), nil
}

// CreateSolidSphere builds a full sphere.
func (t *TransientBRep) CreateSolidSphere(center types.Point, radius float64) (contract.TransientBody, error) {
	b, err := brep.SolidSphere(p3(center), radius, "transient")
	if err != nil {
		return nil, err
	}
	return t.register(b), nil
}

// CreateSolidTorus builds a full torus about the axis.
func (t *TransientBRep) CreateSolidTorus(center types.Point, axis types.Vector, majorRadius, minorRadius float64) (contract.TransientBody, error) {
	b, err := brep.SolidTorus(p3(center), v3(axis), majorRadius, minorRadius, "transient")
	if err != nil {
		return nil, err
	}
	return t.register(b), nil
}

// Copy clones a body into a new handle (lineage derived so keys stay distinct).
func (t *TransientBRep) Copy(body contract.TransientBody) (contract.TransientBody, error) {
	src, err := t.topoOf(body)
	if err != nil {
		return nil, err
	}
	clone, err := CopyTopoBody(src, t.next)
	if err != nil {
		return nil, err
	}
	return t.register(clone), nil
}

// CopyTopoBody clones any kernel body with copy-distinct reference keys.
func CopyTopoBody(src *topo.Body, copyIndex int) (*topo.Body, error) {
	derive := func(l topo.Lineage) topo.Lineage {
		return topo.NewLineage(append(l.Tokens(), topo.Tok("transient", "copy", copyIndex))...)
	}
	return ops.TransformBody(src, math.Identity4(), derive)
}

// Transform maps the body in place by a similarity matrix; scale/shear that
// would break an analytic surface is rejected with the matrix in the error.
func (t *TransientBRep) Transform(body contract.TransientBody, m types.Matrix) error {
	tb, ok := body.(*TransientBody)
	if !ok {
		return fmt.Errorf("bodyapi: Transform needs a registry body, got %T", body)
	}
	moved, err := ops.TransformBody(tb.body, matrix4(m), func(l topo.Lineage) topo.Lineage { return l })
	if err != nil {
		return err
	}
	tb.body = moved
	return nil
}

// DoBoolean combines blank (modified in place) with tool.
func (t *TransientBRep) DoBoolean(blank, tool contract.TransientBody, op types.BooleanType) error {
	bb, ok := blank.(*TransientBody)
	if !ok {
		return fmt.Errorf("bodyapi: DoBoolean blank must be a registry body, got %T", blank)
	}
	toolBody, err := t.topoOf(tool)
	if err != nil {
		return err
	}
	res, err := ops.Boolean(booleanOp(op), bb.body, toolBody)
	if err != nil {
		return err
	}
	bb.body = res
	return nil
}

// booleanOp maps the frozen wire enum onto the kernel operation.
func booleanOp(op types.BooleanType) ops.PartFeatureOperation {
	switch op {
	case types.BooleanUnion:
		return ops.Join
	case types.BooleanIntersect:
		return ops.Intersect
	default:
		return ops.Cut
	}
}

// CreateIntersectionWithPlane sections a body; the curves come back as wires
// on a new transient body.
func (t *TransientBRep) CreateIntersectionWithPlane(body contract.TransientBody, planeOrigin types.Point, planeNormal types.Vector) (contract.TransientBody, error) {
	src, err := t.topoOf(body)
	if err != nil {
		return nil, err
	}
	sec, err := ops.SectionWithPlane(src, p3(planeOrigin), v3(planeNormal), t.q)
	if err != nil {
		return nil, err
	}
	return t.register(sec), nil
}

// CreateFromDefinition compiles a bottom-up graph; issues reject the graph.
func (t *TransientBRep) CreateFromDefinition(def types.BrepBodyDefinition) (contract.TransientBody, []types.BrepDefinitionIssue, error) {
	kdef, issues := DecodeBodyDefinition(def)
	if len(issues) > 0 {
		return nil, issues, nil
	}
	body, compileIssues := brep.CompileSurfaceBodyDefinition(kdef, "definition")
	if len(compileIssues) > 0 {
		return nil, definitionIssues(compileIssues), nil
	}
	return t.register(body), nil, nil
}

// topoOf unwraps a contract body back to its kernel body.
func (t *TransientBRep) topoOf(body contract.TransientBody) (*topo.Body, error) {
	tb, ok := body.(*TransientBody)
	if !ok {
		return nil, fmt.Errorf("bodyapi: expected a registry body, got %T", body)
	}
	return tb.body, nil
}

func p3(p types.Point) math.Point3 {
	return math.P3(math.Scalar(p.X), math.Scalar(p.Y), math.Scalar(p.Z))
}

func v3(v types.Vector) math.Vector3 {
	return math.V3(math.Scalar(v.X), math.Scalar(v.Y), math.Scalar(v.Z))
}

// matrix4 maps the row-major contract matrix onto the kernel matrix (both
// store 16 row-major cells).
func matrix4(m types.Matrix) math.Matrix4 {
	var cells [16]math.Scalar
	for i, c := range m.Cells {
		cells[i] = math.Scalar(c)
	}
	return math.Matrix4FromCells(cells)
}
