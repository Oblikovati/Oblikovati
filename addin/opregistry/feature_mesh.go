// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"
	"os"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

// Mesh reference geometry (M10-F04 PBI-115, #700): an ASCII STL placed as a MeshFeature —
// selectable facet topology that passes the running solid through. Distinct from
// documents.import, which converts a mesh file into B-rep bodies.

const meshSchema = `{
  "type": "object",
  "properties": {
    "path": {"type": "string", "description": "Host-local path of an ASCII STL file to place as mesh reference geometry."},
    "solid": {"type": "boolean", "description": "When true, convert the mesh to a faceted B-rep solid body (one planar face per facet) instead of placing it as reference geometry."}
  },
  "required": ["path"]
}`

func meshDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindMesh, Summary: "Place an ASCII STL as mesh reference geometry.", Schema: json.RawMessage(meshSchema), Apply: applyMesh}
}

func applyMesh(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Mesh](s, raw)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(in.Path)
	if err != nil {
		return nil, fmt.Errorf("mesh: open %q: %w", in.Path, err)
	}
	defer f.Close()
	g, err := feature.ParseSTL(f)
	if err != nil {
		return nil, fmt.Errorf("mesh: parse %q: %w", in.Path, err)
	}
	return recomputeResult(part, addMeshFeature(part.Features(), g, in.Solid))
}

// addMeshFeature places the mesh as a single feature: a faceted B-rep solid when solid is set
// (#492), otherwise a presentation mesh (reference geometry that passes the running solid
// through). The two modes are mutually exclusive.
func addMeshFeature(fs *feature.PartFeatures, g *feature.MeshGeometry, solid bool) *feature.PartFeature {
	meshes := feature.NewMeshFeatures(fs)
	if solid {
		return meshes.AddSolid(g)
	}
	return meshes.Add(g)
}
