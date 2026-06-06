// SPDX-License-Identifier: GPL-2.0-only

// Package topomap maps STEP B-rep topology (CLOSED_SHELL/ADVANCED_FACE/…) onto the
// kernel topo.Builder (from_step) and back (to_step). It is where the STEP sense
// triple — EDGE_CURVE.same_sense, ORIENTED_EDGE.orientation, ADVANCED_FACE.same_sense
// — is composed so every manifold edge ends with two opposite-Reversed uses (the
// invariant ops.Validate checks). See the M17 STEP plan §3 (Topology).
package topomap

import (
	"fmt"

	"oblikovati/kernel/exchange/step/part21"
	"oblikovati/kernel/topo"
)

// SolidFromShell builds a kernel Body from a CLOSED_SHELL (solid=true) or an
// OPEN_SHELL (solid=false). It shares one topo.Vertex per VERTEX_POINT and one
// topo.Edge per EDGE_CURVE so adjacent faces reference the same edge — the
// prerequisite for a closed, validatable solid. Warnings collects non-fatal issues
// (an unsupported surface that was skipped). feat seeds the imported lineage.
func SolidFromShell(g *part21.EntityGraph, shellID int, solid bool, scale float64, feat string) (*topo.Body, []string, error) {
	shell, err := g.Lookup(shellID)
	if err != nil {
		return nil, nil, err
	}
	faceRefs, err := shellFaceRefs(shell)
	if err != nil {
		return nil, nil, err
	}
	a := newAssembler(g, scale, feat, solid)
	for _, faceID := range faceRefs {
		if err := a.addFace(faceID); err != nil {
			return nil, nil, err
		}
	}
	return a.builder.Build(), a.warnings, nil
}

// shellFaceRefs returns the ADVANCED_FACE references of a CLOSED_SHELL/OPEN_SHELL
// (parameters: name, (face…)).
func shellFaceRefs(shell *part21.RawEntity) ([]int, error) {
	if shell.Keyword != "CLOSED_SHELL" && shell.Keyword != "OPEN_SHELL" {
		return nil, fmt.Errorf("topomap: #%d is %s, want CLOSED_SHELL/OPEN_SHELL", shell.ID, shell.Keyword)
	}
	if len(shell.Params) < 2 {
		return nil, fmt.Errorf("topomap: %s #%d missing face list", shell.Keyword, shell.ID)
	}
	return refListValues(shell.Params[1])
}

// refListValues collects the entity references from a list value.
func refListValues(v part21.Value) ([]int, error) {
	items, err := v.AsList()
	if err != nil {
		return nil, err
	}
	out := make([]int, len(items))
	for i, item := range items {
		if out[i], err = item.AsRef(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// assembler holds the per-import sharing maps and the kernel builder.
type assembler struct {
	g        *part21.EntityGraph
	scale    float64
	feat     string
	builder  *topo.Builder
	verts    map[int]*topo.Vertex // VERTEX_POINT id → vertex
	edges    map[int]*topo.Edge   // EDGE_CURVE id → edge (curve runs start→end)
	warnings []string
	nextV    int
	nextE    int
	nextF    int
}

// newAssembler starts an assembler over a fresh builder rooted at feat.
func newAssembler(g *part21.EntityGraph, scale float64, feat string, solid bool) *assembler {
	lineage := topo.NewLineage(topo.Tok(feat, "body", 0))
	return &assembler{
		g: g, scale: scale, feat: feat,
		builder: topo.NewBuilder(solid, lineage),
		verts:   map[int]*topo.Vertex{}, edges: map[int]*topo.Edge{},
	}
}

// warn records a non-fatal issue.
func (a *assembler) warn(format string, args ...any) {
	a.warnings = append(a.warnings, fmt.Sprintf(format, args...))
}
