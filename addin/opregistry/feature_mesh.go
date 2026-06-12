// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"
	"os"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

// Mesh reference geometry (M10-F04 PBI-115, #700): an ASCII STL placed as a MeshFeature —
// selectable facet topology that passes the running solid through. Distinct from
// documents.import, which converts a mesh file into B-rep bodies.

type meshArgs struct {
	Path string `json:"path"`
}

const meshSchema = `{
  "type": "object",
  "properties": {
    "path": {"type": "string", "description": "Host-local path of an ASCII STL file to place as mesh reference geometry."}
  },
  "required": ["path"]
}`

func meshDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "mesh", Summary: "Place an ASCII STL as mesh reference geometry.", Schema: json.RawMessage(meshSchema), Apply: applyMesh}
}

func applyMesh(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in meshArgs
	if err := json.Unmarshal(raw, &in); err != nil {
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
	pf := feature.NewMeshFeatures(part.Features()).Add(g)
	return recomputeResult(part, pf)
}
