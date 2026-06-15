// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// MeshSolid (M20-F15 #492) converts an imported presentation mesh into a faceted B-rep solid:
// one planar face per facet, with shared edges/vertices. Distinct from MeshFeature, which keeps
// the mesh as reference geometry; this feature produces a real solid body usable downstream.

// MeshSolidFeature converts its mesh to a B-rep solid appended as a new body.
type MeshSolidFeature struct {
	geom     *MeshGeometry
	featName string
}

// Geometry returns the source mesh.
func (m *MeshSolidFeature) Geometry() *MeshGeometry { return m.geom }

// Kind implements [Feature].
func (m *MeshSolidFeature) Kind() string { return "mesh-solid" }

// Recompute converts the mesh to a faceted solid and appends it to the running bodies.
func (m *MeshSolidFeature) Recompute(in Input) (Output, error) {
	body := ops.MeshToBRep(m.geom.Vertices, m.geom.Facets, featOr(m.featName, "mesh-solid"))
	if body == nil {
		return Output{}, fmt.Errorf("mesh-solid: mesh has no facets (%d verts, %d facets)", len(m.geom.Vertices), len(m.geom.Facets))
	}
	return Output{Bodies: append(append([]*topo.Body(nil), in.Bodies...), body)}, nil
}

// AddSolid converts mesh geometry to a B-rep solid feature.
func (c *MeshFeatures) AddSolid(g *MeshGeometry) *PartFeature {
	mf := &MeshSolidFeature{geom: g, featName: "MeshSolid"}
	pf := c.engine.Add(mf)
	pf.SetName(c.engine.UniqueName("MeshSolid"))
	mf.featName = pf.name
	return pf
}
