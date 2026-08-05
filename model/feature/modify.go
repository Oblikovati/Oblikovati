// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Modify / direct-edit features operate on whole bodies or picked faces. Combine is
// real in phase A (it is exactly a boolean over two bodies, which the kernel
// already does for the non-overlapping cases); the face-level direct edits
// (move/offset/delete/replace/thicken) and split need the general boolean / tolerant
// machinery (phase C), so they resolve their inputs then defer.

// CombineDefinition combines two running bodies (by index) under a boolean op.
type CombineDefinition struct {
	TargetIndex int
	ToolIndex   int
	Operation   ops.PartFeatureOperation
}

// CombineFeature booleans two bodies in the running state into one result.
type CombineFeature struct{ def *CombineDefinition }

// Definition returns the combine recipe.
func (c *CombineFeature) Definition() *CombineDefinition { return c.def }

// Kind implements [Feature].
func (c *CombineFeature) Kind() string { return "combine" }

// Recompute booleans the target and tool bodies, replacing them with the result.
func (c *CombineFeature) Recompute(in Input) (Output, error) {
	ti, oi := c.def.TargetIndex, c.def.ToolIndex
	if !validIndex(ti, in.Bodies) || !validIndex(oi, in.Bodies) || ti == oi {
		return Output{}, fmt.Errorf("combine: invalid body indices %d,%d (have %d)", ti, oi, len(in.Bodies))
	}
	res, err := ops.BooleanWithDiagnostics(c.def.Operation, in.Bodies[ti], in.Bodies[oi], in.Diag)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceTwo(in.Bodies, ti, oi, res)}, nil
}

func validIndex(i int, bodies []*topo.Body) bool { return i >= 0 && i < len(bodies) }

// replaceTwo returns the bodies with the target/tool removed and the result (if
// non-empty) appended.
func replaceTwo(bodies []*topo.Body, ti, oi int, res *topo.Body) []*topo.Body {
	var out []*topo.Body
	for i, b := range bodies {
		if i != ti && i != oi {
			out = append(out, b)
		}
	}
	if res != nil && len(res.Faces()) > 0 {
		out = append(out, res)
	}
	return out
}

// faceEditFeature is the shared shape of the deferred direct-edit features: they
// resolve picked face keys against the running body, then defer the geometry.
type faceEditFeature struct {
	kind     string
	faceKeys [][]byte
}

func (f *faceEditFeature) Kind() string { return f.kind }
func (f *faceEditFeature) Recompute(in Input) (Output, error) {
	return resolveFacesThenDefer(in, f.faceKeys, f.kind)
}

// FaceKeys returns the reference keys of the faces this direct edit acts on. It lets
// the recipe serialize every face-edit feature uniformly (they share this shape).
func (f *faceEditFeature) FaceKeys() [][]byte { return f.faceKeys }

// SplitFeature is the direct edit whose geometry still defers (phase C).
type SplitFeature struct{ faceEditFeature }

// ThickenFeature turns the running surface (sheet) body into a solid of a wall thickness, or —
// with operation surface — an offset surface (#1876). Direction picks the side(s) offset;
// faceKeys thickens a subset (empty = whole body); walls is Inventor's CreateVerticalSurfaces.
// operation join solidifies; cut/intersect boolean the thickened solid into the running solid.
// Approximation mirrors FaceOffsetFeature's (#331): carried, exact path computed. autoChain /
// autoBlend are accepted for parity but not geometrically applied (selection is explicit).
type ThickenFeature struct {
	thickness     func() float64
	approximation types.FeatureApproximationType
	featName      string
	direction     ops.ThickenDirection
	operation     ops.PartFeatureOperation
	asSurface     bool
	faceKeys      [][]byte
	walls         bool
	autoChain     bool
	autoBlend     bool
}

// Kind implements [Feature].
func (f *ThickenFeature) Kind() string { return "thicken" }

// Thickness returns the wall thickness (for the UI / serialization).
func (f *ThickenFeature) Thickness() float64 { return f.thickness() }

// Approximation returns the requested approximation (zero value = none/exact).
func (f *ThickenFeature) Approximation() types.FeatureApproximationType { return f.approximation }

// SetApproximation records the approximation request (the kernel still computes exact).
func (f *ThickenFeature) SetApproximation(a types.FeatureApproximationType) { f.approximation = a }

// Direction / Operation / AsSurface / FaceKeys / Walls / ChainBlend expose the #1876 options
// for serialization.
func (f *ThickenFeature) Direction() ops.ThickenDirection     { return f.direction }
func (f *ThickenFeature) Operation() ops.PartFeatureOperation { return f.operation }
func (f *ThickenFeature) AsSurface() bool                     { return f.asSurface }
func (f *ThickenFeature) FaceKeys() [][]byte                  { return f.faceKeys }
func (f *ThickenFeature) Walls() bool                         { return f.walls }
func (f *ThickenFeature) ChainBlend() (chain, blend bool)     { return f.autoChain, f.autoBlend }

// SetThickenOptions records the #1876 options on a thicken feature (direction, operation,
// as-surface, face subset, vertical surfaces, and the carried chain/blend flags).
func (f *ThickenFeature) SetThickenOptions(dir ops.ThickenDirection, op ops.PartFeatureOperation, asSurface bool, faceKeys [][]byte, walls, chain, blend bool) {
	f.direction, f.operation, f.asSurface = dir, op, asSurface
	f.faceKeys, f.walls, f.autoChain, f.autoBlend = faceKeys, walls, chain, blend
}

// Recompute thickens the running surface into a solid (or offsets it as a surface); operation
// cut/intersect boolean the solid into the running solid body. See kernel/ops/thicken.go. A
// non-surface / non-thickenable body or a missing boolean target makes the feature go Sick.
func (f *ThickenFeature) Recompute(in Input) (Output, error) {
	surface, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	if f.asSurface {
		return f.recomputeAsSurface(in, surface)
	}
	solid, err := ops.ThickenSolid(surface, f.thickness(), f.direction, f.faceKeys, f.walls)
	if err != nil {
		return Output{}, fmt.Errorf("thicken: %w", err)
	}
	if f.operation == ops.Cut || f.operation == ops.Intersect {
		return thickenBoolean(in, surface, solid, f.operation)
	}
	return Output{Bodies: replaceBody(in.Bodies, surface, solid)}, nil
}

// recomputeAsSurface offsets the (optionally subset) surface into a parallel surface body; a zero
// distance yields a copy (Inventor's Thicken surface, Distance 0).
func (f *ThickenFeature) recomputeAsSurface(in Input, surface *topo.Body) (Output, error) {
	src := surface
	if len(f.faceKeys) > 0 {
		kept, err := ops.DropFaces(surface, f.faceKeys, true)
		if err != nil {
			return Output{}, fmt.Errorf("thicken surface: %w", err)
		}
		src = kept
	}
	d := f.thickness()
	if f.direction == ops.ThickenNegative {
		d = -d
	}
	result, err := ops.OffsetSurface(src, d, f.featName)
	if err != nil {
		return Output{}, fmt.Errorf("thicken surface: %w", err)
	}
	return Output{Bodies: replaceBody(in.Bodies, surface, result)}, nil
}

// thickenBoolean cuts/intersects the thickened solid into the most recent running solid body
// (the surface is consumed). A missing target solid makes the feature go Sick.
func thickenBoolean(in Input, surface, solid *topo.Body, op ops.PartFeatureOperation) (Output, error) {
	target := lastSolidExcept(in.Bodies, surface)
	if target == nil {
		return Output{}, fmt.Errorf("thicken %s: no solid body to modify", op)
	}
	result, err := ops.Boolean(op, target, solid)
	if err != nil {
		return Output{}, fmt.Errorf("thicken %s: %w", op, err)
	}
	bodies := make([]*topo.Body, 0, len(in.Bodies))
	for _, b := range in.Bodies {
		switch b {
		case surface: // consumed by the thicken
		case target:
			bodies = append(bodies, result)
		default:
			bodies = append(bodies, b)
		}
	}
	return Output{Bodies: bodies}, nil
}

// lastSolidExcept returns the most recent solid body that is not skip, or nil.
func lastSolidExcept(bodies []*topo.Body, skip *topo.Body) *topo.Body {
	for i := len(bodies) - 1; i >= 0; i-- {
		if bodies[i] != skip && bodies[i].IsSolid() {
			return bodies[i]
		}
	}
	return nil
}

// ReplaceFaceFeature replaces the picked faces' surface with that of a target face.
type ReplaceFaceFeature struct {
	faceEditFeature
	targetKey    []byte       // legacy: a same-body face key, re-resolved each recompute (associative)
	targetPlanes []geom.Plane // #1886: frozen new-face / work-plane target planes (from the router)
}

// TargetKey returns the reference key of the legacy single same-body target face.
func (f *ReplaceFaceFeature) TargetKey() []byte { return f.targetKey }

// TargetPlanes returns the frozen new-face target planes (#1886), empty for the legacy path.
func (f *ReplaceFaceFeature) TargetPlanes() []geom.Plane { return f.targetPlanes }

// Recompute replaces the picked faces with their target plane(s) on the running body (see
// kernel/ops/replace_face.go). The #1886 targetPlanes path assigns each picked face to its nearest
// frozen target (work plane or new face, possibly cross-body); the legacy path re-resolves a single
// same-body target face each recompute. A lost picked or target face makes the feature go Sick.
func (f *ReplaceFaceFeature) Recompute(in Input) (Output, error) {
	return retopoFacesBody(in, f.faceKeys, f.kind, func(b *topo.Body, keys [][]byte) (*topo.Body, error) {
		if len(f.targetPlanes) > 0 {
			return ops.ReplaceFacesMulti(b, keys, f.targetPlanes)
		}
		target, ok := ops.PlaneOfFace(b, f.targetKey)
		if !ok {
			return nil, fmt.Errorf("replace-face: target face reference lost")
		}
		return ops.ReplaceFaces(b, keys, target)
	})
}

// DeleteFaceFeature removes the picked faces. Heal (default false, Inventor parity #1884) extends
// the neighbouring faces to close the opening; heal=false leaves the body open (a surface). If the
// picked faces sit on an internal void shell instead, the whole void is removed (mass restored).
type DeleteFaceFeature struct {
	faceEditFeature
	heal bool
}

// Heal reports whether the openings are closed by extending neighbours (for serialization).
func (f *DeleteFaceFeature) Heal() bool { return f.heal }

// Recompute deletes the picked faces from the running body. Faces on an internal void shell drop
// that void (restoring mass, see kernel/ops/delete_void.go); otherwise heal extends neighbours to
// close the gap (kernel/ops/delete_face.go) and heal=false leaves it open (kernel/ops/drop_faces.go).
// A non-healable selection or lost key makes the feature go Sick.
func (f *DeleteFaceFeature) Recompute(in Input) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	q := ops.DefaultQuality()
	var result *topo.Body
	switch {
	case ops.FacesOnVoidShell(body, f.faceKeys, q):
		result, err = ops.RemoveVoidShellByFaces(body, f.faceKeys, q)
	case f.heal:
		result, err = ops.DeleteFaces(body, f.faceKeys)
	default:
		result, err = ops.DropFaces(body, f.faceKeys, false)
	}
	if err != nil {
		return Output{}, fmt.Errorf("%s: %w", f.kind, err)
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
}

// MoveFaceFeature translates the picked faces by a vector — or, in rotate mode (#331),
// rotates them about an axis — retrimming the neighbours.
type MoveFaceFeature struct {
	faceEditFeature
	translation math.Vector3
	axisPoint   math.Point3
	axisDir     math.Vector3 // zero = translate mode
	angle       func() float64
}

// Translation returns the move-face displacement (for the UI / serialization).
func (f *MoveFaceFeature) Translation() math.Vector3 { return f.translation }

// Rotation returns the rotate-mode axis and angle; rotating reports false in translate mode.
func (f *MoveFaceFeature) Rotation() (point math.Point3, dir math.Vector3, angle float64, rotating bool) {
	if f.angle == nil {
		return math.Point3{}, math.Vector3{}, 0, false
	}
	return f.axisPoint, f.axisDir, f.angle(), true
}

// Recompute moves (or rotates) the picked faces on the running body (see
// kernel/ops/move_face.go).
func (f *MoveFaceFeature) Recompute(in Input) (Output, error) {
	return retopoFacesBody(in, f.faceKeys, f.kind, func(b *topo.Body, keys [][]byte) (*topo.Body, error) {
		if f.angle == nil {
			return ops.MoveFaces(b, keys, f.translation)
		}
		dir, err := math.UnitVector3FromVector(f.axisDir)
		if err != nil {
			return nil, fmt.Errorf("rotate axis %v is degenerate", f.axisDir)
		}
		return ops.RotateFaces(b, keys, f.axisPoint, dir, f.angle())
	})
}

// FaceOffsetFeature moves the picked faces along their own normals by a distance.
// Approximation is the #331 parity input: the kernel computes the EXACT offset, which
// satisfies every approximation bound, so the choice is carried for the API/UI without
// changing geometry.
type FaceOffsetFeature struct {
	faceEditFeature
	distance      func() float64
	approximation types.FeatureApproximationType
}

// Distance returns the face-offset distance (for the UI / serialization).
func (f *FaceOffsetFeature) Distance() float64 { return f.distance() }

// Approximation returns the requested approximation (zero value = none/exact).
func (f *FaceOffsetFeature) Approximation() types.FeatureApproximationType { return f.approximation }

// Recompute offsets the picked faces on the running body (see kernel/ops/move_face.go).
func (f *FaceOffsetFeature) Recompute(in Input) (Output, error) {
	return retopoFacesBody(in, f.faceKeys, f.kind, func(b *topo.Body, keys [][]byte) (*topo.Body, error) {
		return ops.OffsetFaces(b, keys, f.distance())
	})
}

// retopoFacesBody applies a face retopology op to the running body and replaces it; a lost
// key (surfaced by the op) makes the feature go Sick.
func retopoFacesBody(in Input, keys [][]byte, feat string, op func(*topo.Body, [][]byte) (*topo.Body, error)) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	result, err := op(body, keys)
	if err != nil {
		return Output{}, fmt.Errorf("%s: %w", feat, err)
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
}

// ModifyFeatures adds modify/direct-edit features into the engine.
type ModifyFeatures struct{ engine *PartFeatures }

// NewModifyFeatures binds the collection to an engine.
func NewModifyFeatures(engine *PartFeatures) *ModifyFeatures { return &ModifyFeatures{engine} }

// AddCombine booleans two running bodies (by index) under op.
func (c *ModifyFeatures) AddCombine(targetIndex, toolIndex int, op ops.PartFeatureOperation) *PartFeature {
	return c.engine.Add(&CombineFeature{def: &CombineDefinition{TargetIndex: targetIndex, ToolIndex: toolIndex, Operation: op}})
}

// AddSplit/AddMoveFace/AddFaceOffset/AddDeleteFace/AddReplaceFace/AddThicken add the
// deferred direct edits over the given face keys.
func (c *ModifyFeatures) AddSplit(faceKeys [][]byte) *PartFeature {
	return c.engine.Add(&SplitFeature{faceEditFeature{kind: "split", faceKeys: faceKeys}})
}

func (c *ModifyFeatures) AddMoveFace(faceKeys [][]byte, translation math.Vector3) *PartFeature {
	return c.engine.Add(&MoveFaceFeature{faceEditFeature: faceEditFeature{kind: "move-face", faceKeys: faceKeys}, translation: translation})
}

// AddMoveFaceRotate is the rotate arm of move-face (#331): the picked faces rotate by angle
// about the axis (point + direction).
func (c *ModifyFeatures) AddMoveFaceRotate(faceKeys [][]byte, axisPoint math.Point3, axisDir math.Vector3, angle func() float64) *PartFeature {
	return c.engine.Add(&MoveFaceFeature{
		faceEditFeature: faceEditFeature{kind: "move-face", faceKeys: faceKeys},
		axisPoint:       axisPoint, axisDir: axisDir, angle: angle,
	})
}

// AddFaceOffsetApprox is AddFaceOffsetFn carrying the #331 approximation request (the kernel
// computes the exact offset, which satisfies every approximation bound).
func (c *ModifyFeatures) AddFaceOffsetApprox(faceKeys [][]byte, distance func() float64, approx types.FeatureApproximationType) *PartFeature {
	return c.engine.Add(&FaceOffsetFeature{
		faceEditFeature: faceEditFeature{kind: "face-offset", faceKeys: faceKeys},
		distance:        distance, approximation: approx,
	})
}

// AddDeleteFace deletes the picked faces. heal=true extends the neighbouring faces to close the
// opening; heal=false leaves the body open. Faces on an internal void shell drop that void (#1884).
func (c *ModifyFeatures) AddDeleteFace(faceKeys [][]byte, heal bool) *PartFeature {
	return c.engine.Add(&DeleteFaceFeature{faceEditFeature: faceEditFeature{kind: "delete-face", faceKeys: faceKeys}, heal: heal})
}

func (c *ModifyFeatures) AddReplaceFace(faceKeys [][]byte, targetKey []byte) *PartFeature {
	return c.engine.Add(&ReplaceFaceFeature{faceEditFeature: faceEditFeature{kind: "replace-face", faceKeys: faceKeys}, targetKey: targetKey})
}

// AddReplaceFacePlanes replaces the picked faces with a set of frozen target planes (#1886) — work
// planes and/or new faces, possibly from other bodies. Each picked face is assigned to its nearest
// target at recompute.
func (c *ModifyFeatures) AddReplaceFacePlanes(faceKeys [][]byte, targets []geom.Plane) *PartFeature {
	return c.engine.Add(&ReplaceFaceFeature{faceEditFeature: faceEditFeature{kind: "replace-face", faceKeys: faceKeys}, targetPlanes: targets})
}

// AddThicken thickens the running surface body into a solid of the given wall thickness.
func (c *ModifyFeatures) AddThicken(thickness float64) *PartFeature {
	return c.AddThickenFn(constFloat(thickness))
}

// AddThickenFn is AddThicken with a live (parameter-driven) thickness. Options default to the
// whole-body, symmetric, join, walls-on thicken; the router/UI refine them via SetThickenOptions.
func (c *ModifyFeatures) AddThickenFn(thickness func() float64) *PartFeature {
	tf := &ThickenFeature{thickness: thickness, walls: true, featName: "Thicken"}
	pf := c.engine.Add(tf)
	tf.featName = pf.name
	return pf
}
