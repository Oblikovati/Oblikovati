// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Bottom-up B-rep construction (M07-F05, Oblikovati/Oblikovati#628): the
// reference SurfaceBodyDefinition graph — vertices, edges, faces with
// edge-use loops, shells grouped into lumps, plus wire edges — compiled into
// a topo.Body through the Builder. Construction problems are collected per
// definition (the reference CreateTransientSurfaceBody errors map), not
// aborted at the first.

// VertexDefinition is one definition-graph vertex.
type VertexDefinition struct {
	Position      math.Point3
	AssociativeID int
}

// EdgeDefinition is one edge: its model-space curve and endpoint vertex
// indices (into the definition's Vertices).
type EdgeDefinition struct {
	Curve         geom.Curve3
	StartVertex   int
	EndVertex     int
	AssociativeID int
}

// EdgeUseDefinition is one oriented edge use within a loop (Opposed =
// traversed against the edge's natural direction).
type EdgeUseDefinition struct {
	Edge    int
	Opposed bool
}

// EdgeLoopDefinition is one boundary loop; a face's FIRST loop is its outer.
type EdgeLoopDefinition struct {
	Uses []EdgeUseDefinition
}

// FaceDefinition is one face: surface geometry, loops, and whether the
// material side opposes the surface normal.
type FaceDefinition struct {
	Surface       geom.Surface
	ParamReversed bool
	Loops         []EdgeLoopDefinition
	AssociativeID int
}

// FaceShellDefinition groups face indices (into the definition's Faces).
type FaceShellDefinition struct {
	Faces []int
}

// WireDefinition is an ordered chain of edge indices carried without faces.
type WireDefinition struct {
	Edges []int
}

// LumpDefinition groups shells and wires (the body's connected regions).
type LumpDefinition struct {
	Shells []FaceShellDefinition
	Wires  []WireDefinition
}

// SurfaceBodyDefinition is the whole graph.
type SurfaceBodyDefinition struct {
	Vertices []VertexDefinition
	Edges    []EdgeDefinition
	Faces    []FaceDefinition
	Lumps    []LumpDefinition
	Solid    bool
}

// DefinitionIssue is one construction problem, addressed by its graph path.
type DefinitionIssue struct {
	Path    string
	Problem string
}

func issuef(issues []DefinitionIssue, path, format string, args ...interface{}) []DefinitionIssue {
	return append(issues, DefinitionIssue{Path: path, Problem: fmt.Sprintf(format, args...)})
}

// CompileSurfaceBodyDefinition builds the body. A non-empty issue list means
// the graph was rejected (no body); the topological validity of an
// issue-free result is the caller's ops.Validate pass.
//
// Example: body, issues := brep.CompileSurfaceBodyDefinition(def, "addin")
func CompileSurfaceBodyDefinition(def SurfaceBodyDefinition, feat string) (*topo.Body, []DefinitionIssue) {
	issues := validateDefinitionIndices(def)
	if len(issues) > 0 {
		return nil, issues
	}
	bld := topo.NewBuilder(def.Solid, topo.NewLineage(topo.Tok(feat, "body", 0)))
	verts := compileVertices(bld, def, feat)
	edges, issues := compileEdges(bld, def, verts, feat)
	if len(issues) > 0 {
		return nil, issues
	}
	compileFaces(bld, def, edges, feat)
	body := bld.Build()
	compileWires(body, def, edges, feat)
	return body, nil
}

// validateDefinitionIndices checks every cross-reference before building.
func validateDefinitionIndices(def SurfaceBodyDefinition) []DefinitionIssue {
	var issues []DefinitionIssue
	for i, e := range def.Edges {
		if e.StartVertex < 0 || e.StartVertex >= len(def.Vertices) || e.EndVertex < 0 || e.EndVertex >= len(def.Vertices) {
			issues = issuef(issues, fmt.Sprintf(edgeIssueLabel, i),
				"vertex indices (%d, %d) out of range (have %d vertices)", e.StartVertex, e.EndVertex, len(def.Vertices))
		}
		if e.Curve == nil {
			issues = issuef(issues, fmt.Sprintf(edgeIssueLabel, i), "edge has no curve")
		}
	}
	issues = append(issues, validateFaceIndices(def)...)
	issues = append(issues, validateLumpIndices(def)...)
	return issues
}

func validateFaceIndices(def SurfaceBodyDefinition) []DefinitionIssue {
	var issues []DefinitionIssue
	for i, f := range def.Faces {
		if f.Surface == nil {
			issues = issuef(issues, fmt.Sprintf("faces[%d]", i), "face has no surface")
		}
		for j, l := range f.Loops {
			if len(l.Uses) == 0 {
				issues = issuef(issues, fmt.Sprintf("faces[%d].loops[%d]", i, j), "loop has no edge uses")
			}
			for k, u := range l.Uses {
				if u.Edge < 0 || u.Edge >= len(def.Edges) {
					issues = issuef(issues, fmt.Sprintf("faces[%d].loops[%d].uses[%d]", i, j, k),
						"edge index %d out of range (have %d edges)", u.Edge, len(def.Edges))
				}
			}
		}
	}
	return issues
}

func validateLumpIndices(def SurfaceBodyDefinition) []DefinitionIssue {
	var issues []DefinitionIssue
	for i, lump := range def.Lumps {
		for j, sh := range lump.Shells {
			for k, fi := range sh.Faces {
				if fi < 0 || fi >= len(def.Faces) {
					issues = issuef(issues, fmt.Sprintf("lumps[%d].shells[%d].faces[%d]", i, j, k),
						"face index %d out of range (have %d faces)", fi, len(def.Faces))
				}
			}
		}
		for j, w := range lump.Wires {
			for k, ei := range w.Edges {
				if ei < 0 || ei >= len(def.Edges) {
					issues = issuef(issues, fmt.Sprintf("lumps[%d].wires[%d].edges[%d]", i, j, k),
						"edge index %d out of range (have %d edges)", ei, len(def.Edges))
				}
			}
		}
	}
	return issues
}

func compileVertices(bld *topo.Builder, def SurfaceBodyDefinition, feat string) []*topo.Vertex {
	out := make([]*topo.Vertex, len(def.Vertices))
	for i, v := range def.Vertices {
		out[i] = bld.AddVertex(v.Position, topo.NewLineage(topo.Tok(feat, "vertex", associativeOr(v.AssociativeID, i))))
	}
	return out
}

// compileEdges builds the edges, checking each curve's ends actually land on
// its declared vertices (the classic hand-built-graph mistake).
func compileEdges(bld *topo.Builder, def SurfaceBodyDefinition, verts []*topo.Vertex, feat string) ([]*topo.Edge, []DefinitionIssue) {
	const endTol = 1e-6
	var issues []DefinitionIssue
	out := make([]*topo.Edge, len(def.Edges))
	for i, e := range def.Edges {
		lo, hi := e.Curve.Domain()
		s, t := e.Curve.PointAt(lo), e.Curve.PointAt(hi)
		sv, ev := verts[e.StartVertex].Point(), verts[e.EndVertex].Point()
		if float64(s.DistanceTo(sv)) > endTol || float64(t.DistanceTo(ev)) > endTol {
			issues = issuef(issues, fmt.Sprintf(edgeIssueLabel, i),
				"curve ends %v→%v do not meet vertices %v→%v (tolerance %g)", s, t, sv, ev, endTol)
			continue
		}
		out[i] = bld.AddEdge(e.Curve, verts[e.StartVertex], verts[e.EndVertex],
			topo.NewLineage(topo.Tok(feat, "edge", associativeOr(e.AssociativeID, i))))
	}
	return out, issues
}

// compileFaces builds the faces (index validity was established up front, so
// this stage cannot fail).
func compileFaces(bld *topo.Builder, def SurfaceBodyDefinition, edges []*topo.Edge, feat string) {
	for i, f := range def.Faces {
		specs := make([]topo.LoopSpec, len(f.Loops))
		for j, l := range f.Loops {
			uses := make([]topo.Use, len(l.Uses))
			for k, u := range l.Uses {
				uses[k] = topo.Use{Edge: edges[u.Edge], Reversed: u.Opposed}
			}
			if j == 0 {
				specs[j] = topo.OuterLoop(uses...)
			} else {
				specs[j] = topo.InnerLoop(uses...)
			}
		}
		lin := topo.NewLineage(topo.Tok(feat, "face", associativeOr(f.AssociativeID, i)))
		if f.ParamReversed {
			bld.AddReversedFace(f.Surface, lin, specs...)
		} else {
			bld.AddFace(f.Surface, lin, specs...)
		}
	}
}

func compileWires(body *topo.Body, def SurfaceBodyDefinition, edges []*topo.Edge, feat string) {
	wi := 0
	for _, lump := range def.Lumps {
		for _, w := range lump.Wires {
			uses := make([]topo.Use, len(w.Edges))
			for k, ei := range w.Edges {
				uses[k] = topo.Fwd(edges[ei])
			}
			body.AttachWire(topo.NewLineage(topo.Tok(feat, "wire", wi)), uses)
			wi++
		}
	}
}

// associativeOr prefers the caller's associative id, falling back to the
// definition index, so reference keys are caller-stable when ids are given.
func associativeOr(id, fallback int) int {
	if id != 0 {
		return id
	}
	return fallback
}
