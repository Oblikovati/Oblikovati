// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

// hull replaces the part's running solids with their single convex hull — OpenSCAD's hull().
// It takes no arguments: every running solid body is hulled into one (build the primitives as
// separate bodies with operation "new", then hull them — e.g. two cylinders → a capsule).

const hullSchema = `{
  "type": "object",
  "properties": {},
  "additionalProperties": false
}`

func hullDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindHull,
		Summary: "Convex hull of the part's running solids into one body (OpenSCAD hull()).",
		Schema:  json.RawMessage(hullSchema),
		Apply:   applyHull,
	}
}

func applyHull(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	pf := feature.NewHullFeatures(part.Features()).Add()
	return recomputeResult(part, pf)
}
