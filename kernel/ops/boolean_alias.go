// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops/boolean"
	"oblikovati.org/kernel/ops/meshbrep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The boolean engine moved to kernel/ops/boolean (#2204) so a change to the mesh-arrangement
// or reconstruction path no longer rebuilds and re-tests every package that merely names an
// operation. These are the names call sites already use; new code may import the narrow
// package directly.

// PartFeatureOperation is the boolean a feature applies to existing material.
// See [boolean.PartFeatureOperation].
type PartFeatureOperation = boolean.PartFeatureOperation

const (
	// Join adds material (A ∪ B).
	Join = boolean.Join
	// Cut removes the tool from the target (A − B).
	Cut = boolean.Cut
	// Intersect keeps only the common material (A ∩ B).
	Intersect = boolean.Intersect
	// NewBody creates a separate body rather than combining.
	NewBody = boolean.NewBody
	// Surface builds an open sheet body and applies no boolean at all.
	Surface = boolean.Surface
)

// Boolean combines two bodies under op. See [boolean.Boolean].
func Boolean(op PartFeatureOperation, target, tool *topo.Body) (*topo.Body, error) {
	return boolean.Boolean(op, target, tool)
}

// BooleanWithDiagnostics is Boolean with a recorder for the defects it raises.
// See [boolean.BooleanWithDiagnostics].
func BooleanWithDiagnostics(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, error) {
	return boolean.BooleanWithDiagnostics(op, target, tool, rec)
}

// CurvedBoolean is the curved-surface entry point. See [boolean.CurvedBoolean].
func CurvedBoolean(op PartFeatureOperation, target, tool *topo.Body) (*topo.Body, bool) {
	return boolean.CurvedBoolean(op, target, tool)
}

// CurvedBooleanWithDiagnostics is CurvedBoolean with a recorder.
// See [boolean.CurvedBooleanWithDiagnostics].
func CurvedBooleanWithDiagnostics(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	return boolean.CurvedBooleanWithDiagnostics(op, target, tool, rec)
}

// PointInsideBody reports whether p is strictly inside a solid body.
// See [boolean.PointInsideBody].
func PointInsideBody(b *topo.Body, p math.Point3) bool { return boolean.PointInsideBody(b, p) }

// MeshToBRep converts a closed welded mesh into a faceted B-rep solid. See [meshbrep.MeshToBRep].
func MeshToBRep(verts []math.Point3, facets [][]int, feat string) *topo.Body {
	return meshbrep.MeshToBRep(verts, facets, feat)
}

// The defect codes the boolean raises, re-exported so consumers match on one name.
const (
	CodeBooleanAnalyticFaceted      diag.Code = boolean.CodeBooleanAnalyticFaceted
	CodeBooleanAnalyticVolumeReject diag.Code = boolean.CodeBooleanAnalyticVolumeReject
	CodeBooleanAnalyticFaceReject   diag.Code = boolean.CodeBooleanAnalyticFaceReject
	CodeBooleanNoExactCurvedPath    diag.Code = boolean.CodeBooleanNoExactCurvedPath
)

// ErrUnmodelledBoolean is the boolean's refusal: the operands meet in a configuration no exact path
// models. See [boolean.ErrUnmodelledBoolean].
var ErrUnmodelledBoolean = boolean.ErrUnmodelledBoolean
