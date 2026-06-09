// SPDX-License-Identifier: GPL-2.0-only

package topomap

import (
	"oblikovati.org/kernel/exchange/step/geommap"
	"oblikovati.org/kernel/exchange/step/part21"
	"oblikovati.org/kernel/topo"
)

// disassembler emits a kernel Body as STEP topology, sharing one STEP entity per
// kernel vertex/edge (keyed on the kernel pointer) so adjacent faces reference the
// same EDGE_CURVE — the dual of the import-side assembler.
type disassembler struct {
	emit  *geommap.Emitter
	verts map[*topo.Vertex]int
	edges map[*topo.Edge]int
}

// BodyToStep emits body as a MANIFOLD_SOLID_BREP (solid) or surface model into the
// emitter's writer and returns the BREP entity id. It first emits all faces (which
// emit their bounding vertices/edges on demand), then the closed shell and brep.
func BodyToStep(emit *geommap.Emitter, body *topo.Body) (int, error) {
	d := &disassembler{emit: emit, verts: map[*topo.Vertex]int{}, edges: map[*topo.Edge]int{}}
	faceIDs, err := d.emitFaces(body)
	if err != nil {
		return 0, err
	}
	shell := d.emitShell(body.IsSolid(), faceIDs)
	return d.emitBrep(body.IsSolid(), shell), nil
}

// emitFaces emits every face of the body's shells, returning their entity ids.
func (d *disassembler) emitFaces(body *topo.Body) ([]int, error) {
	var ids []int
	for _, shell := range body.Shells() {
		for _, f := range shell.Faces() {
			id, err := d.emitFace(f)
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// emitShell emits a CLOSED_SHELL (solid) or OPEN_SHELL (surface) over the faces.
func (d *disassembler) emitShell(solid bool, faceIDs []int) int {
	keyword := "OPEN_SHELL"
	if solid {
		keyword = "CLOSED_SHELL"
	}
	return d.emit.Writer().Add(keyword, part21.QuoteString(""), refList(faceIDs))
}

// emitBrep wraps a shell as a MANIFOLD_SOLID_BREP (solid) or a
// SHELL_BASED_SURFACE_MODEL (surface body).
func (d *disassembler) emitBrep(solid bool, shellID int) int {
	w := d.emit.Writer()
	if solid {
		return w.Add("MANIFOLD_SOLID_BREP", part21.QuoteString(""), part21.Ref(shellID))
	}
	return w.Add("SHELL_BASED_SURFACE_MODEL", part21.QuoteString(""), refList([]int{shellID}))
}

// refList formats a list of entity ids as a STEP reference list (#a,#b,…).
func refList(ids []int) string {
	refs := make([]string, len(ids))
	for i, id := range ids {
		refs[i] = part21.Ref(id)
	}
	return part21.FormatList(refs...)
}
