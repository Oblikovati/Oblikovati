// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
)

// bodyFaceEvaluate serves wire.MethodBodyFaceEvaluate: a batched surface evaluation of one
// face addressed by reference key. It is the out-of-process projection of the kernel surface
// evaluator (kernel/topo) — a caller sampling a face densely (surface-following toolpaths,
// point projection) gets all points back in one reply instead of one call per point.
func bodyFaceEvaluate(_ *app.Session, part *compdef.PartComponentDefinition, in wire.FaceEvaluateArgs) (wire.FaceEvaluateResult, error) {
	b, err := bodyAt(part, in.BodyIndex)
	if err != nil {
		return wire.FaceEvaluateResult{}, err
	}
	f, ok := b.FindFaceByKey([]byte(in.FaceKey))
	if !ok {
		return wire.FaceEvaluateResult{}, fmt.Errorf("no face with key %q on body %d", in.FaceKey, in.BodyIndex)
	}
	eval := topo.NewFaceEvaluator(f).SurfaceEvaluator
	out, err := evaluateFaceSurface(eval, in.Mode, in.Inputs)
	if err != nil {
		return wire.FaceEvaluateResult{}, err
	}
	uLo, uHi, vLo, vHi := eval.ParamRange()
	out.ParamRange = []float64{uLo, vLo, uHi, vHi}
	return out, nil
}

// evaluateFaceSurface runs the requested batched query over the inputs, filling only the
// result arrays the mode produces. Parametric modes read (u, v) pairs; closestPoint reads
// (x, y, z) point triples and projects each onto the surface.
func evaluateFaceSurface(e topo.SurfaceEvaluator, mode string, inputs []float64) (wire.FaceEvaluateResult, error) {
	var out wire.FaceEvaluateResult
	switch mode {
	case wire.FaceEvalPointAtParam:
		forEachParam(inputs, func(u, v float64) { out.Points = appendPoint(out.Points, e.PointAt(u, v)) })
	case wire.FaceEvalNormalAtParam:
		forEachParam(inputs, func(u, v float64) {
			out.Points = appendPoint(out.Points, e.PointAt(u, v))
			out.Normals = appendVector(out.Normals, e.NormalAt(u, v))
		})
	case wire.FaceEvalTangents:
		forEachParam(inputs, func(u, v float64) {
			du, dv := e.TangentsAt(u, v)
			out.Points = appendPoint(out.Points, e.PointAt(u, v))
			out.UTangents = appendVector(out.UTangents, du)
			out.VTangents = appendVector(out.VTangents, dv)
		})
	case wire.FaceEvalClosestPoint:
		forEachPoint(inputs, func(p math.Point3) {
			u, v := e.ClosestParam(p)
			out.Points = appendPoint(out.Points, e.PointAt(u, v))
			out.UVs = append(out.UVs, u, v)
		})
	default:
		return out, fmt.Errorf("unknown face-evaluate mode %q (want %q, %q, %q or %q)",
			mode, wire.FaceEvalPointAtParam, wire.FaceEvalNormalAtParam, wire.FaceEvalTangents, wire.FaceEvalClosestPoint)
	}
	return out, nil
}

// forEachParam invokes fn for each (u, v) pair in a flat parameter array (a trailing
// unpaired value is ignored).
func forEachParam(flat []float64, fn func(u, v float64)) {
	for i := 0; i+1 < len(flat); i += 2 {
		fn(flat[i], flat[i+1])
	}
}

// forEachPoint invokes fn for each (x, y, z) triple in a flat point array.
func forEachPoint(flat []float64, fn func(p math.Point3)) {
	for i := 0; i+2 < len(flat); i += 3 {
		fn(math.P3(flat[i], flat[i+1], flat[i+2]))
	}
}

// appendPoint flattens a point's (x, y, z) onto a result array.
func appendPoint(dst []float64, p math.Point3) []float64 { return append(dst, p.X, p.Y, p.Z) }

// appendVector flattens a vector's (x, y, z) onto a result array.
func appendVector(dst []float64, v math.Vector3) []float64 { return append(dst, v.X, v.Y, v.Z) }
