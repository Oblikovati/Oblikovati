// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
)

// tetraDefinition is the unit tetrahedron as a bottom-up definition graph —
// 4 vertices, 6 edges, 4 faces with consistently outward loops.
func tetraDefinition() brep.SurfaceBodyDefinition {
	p := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0), math.P3(0, 0, 1)}
	var def brep.SurfaceBodyDefinition
	def.Solid = true
	for i, pt := range p {
		def.Vertices = append(def.Vertices, brep.VertexDefinition{Position: pt, AssociativeID: 100 + i})
	}
	pairs := [][2]int{{0, 1}, {0, 2}, {0, 3}, {1, 2}, {1, 3}, {2, 3}}
	for _, pr := range pairs {
		def.Edges = append(def.Edges, brep.EdgeDefinition{
			Curve: geom.NewLineSegment(p[pr[0]], p[pr[1]]), StartVertex: pr[0], EndVertex: pr[1],
		})
	}
	// Outward loops — the same windings as ops_test's proven tetra builder
	// (each shared edge traversed once Fwd, once Opposed).
	use := func(e int, opp bool) brep.EdgeUseDefinition { return brep.EdgeUseDefinition{Edge: e, Opposed: opp} }
	loops := [][]brep.EdgeUseDefinition{
		{use(1, false), use(3, true), use(0, true)},  // bottom (z=0), outward −Z
		{use(0, false), use(4, false), use(2, true)}, // y=0 side, outward −Y
		{use(2, false), use(5, true), use(1, true)},  // x=0 side, outward −X
		{use(3, false), use(5, false), use(4, true)}, // slanted, outward (1,1,1)
	}
	normals := []math.Vector3{math.V3(0, 0, -1), math.V3(0, -1, 0), math.V3(-1, 0, 0), math.V3(1, 1, 1)}
	origins := []math.Point3{p[0], p[0], p[0], p[1]}
	for i := range loops {
		surf, _ := geom.NewPlane(origins[i], normals[i])
		def.Faces = append(def.Faces, brep.FaceDefinition{Surface: surf, Loops: []brep.EdgeLoopDefinition{{Uses: loops[i]}}})
	}
	def.Lumps = []brep.LumpDefinition{{Shells: []brep.FaceShellDefinition{{Faces: []int{0, 1, 2, 3}}}}}
	return def
}

// TestCompileDefinitionTetra: the graph compiles into a valid solid of the
// analytic volume 1/6, with caller-stable vertex keys.
func TestCompileDefinitionTetra(t *testing.T) {
	t.Parallel()
	body, issues := brep.CompileSurfaceBodyDefinition(tetraDefinition(), "addin")
	if len(issues) > 0 {
		t.Fatalf("compile issues: %+v", issues)
	}
	if r := ops.Validate(body); !r.Valid || !r.Closed {
		t.Fatalf("compiled tetra invalid: %+v", r.Issues)
	}
	if v := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; stdmath.Abs(v-1.0/6) > 1e-9 {
		t.Errorf("compiled tetra volume = %g, want 1/6", v)
	}
	if len(body.Vertices()) != 4 || len(body.Edges()) != 6 || len(body.Faces()) != 4 {
		t.Errorf("topology = %d/%d/%d, want 4/6/4", len(body.Vertices()), len(body.Edges()), len(body.Faces()))
	}
}

// TestCompileDefinitionReportsProblems: bad indices and detached curves are
// reported per definition with their graph path, and no body is built.
func TestCompileDefinitionReportsProblems(t *testing.T) {
	t.Parallel()
	def := tetraDefinition()
	def.Edges[2].EndVertex = 99 // out of range
	body, issues := brep.CompileSurfaceBodyDefinition(def, "addin")
	if body != nil || len(issues) == 0 {
		t.Fatal("an out-of-range vertex index must reject the graph")
	}
	if issues[0].Path != "edges[2]" || !strings.Contains(issues[0].Problem, "99") {
		t.Errorf("issue = %+v, want edges[2] naming index 99", issues[0])
	}

	def2 := tetraDefinition()
	def2.Edges[0].Curve = geom.NewLineSegment(math.P3(5, 5, 5), math.P3(6, 6, 6)) // curve off its vertices
	if body, issues := brep.CompileSurfaceBodyDefinition(def2, "addin"); body != nil || len(issues) == 0 {
		t.Error("a curve detached from its vertices must reject the graph")
	}
}

// TestCompileDefinitionWires: wire definitions land as body wires.
func TestCompileDefinitionWires(t *testing.T) {
	t.Parallel()
	def := tetraDefinition()
	def.Faces = nil
	def.Solid = false
	def.Lumps = []brep.LumpDefinition{{Wires: []brep.WireDefinition{{Edges: []int{0, 3}}}}}
	body, issues := brep.CompileSurfaceBodyDefinition(def, "addin")
	if len(issues) > 0 {
		t.Fatalf("compile issues: %+v", issues)
	}
	if len(body.Wires()) != 1 || len(body.Wires()[0].Edges()) != 2 {
		t.Errorf("wire body has %d wires, want 1 with 2 edges", len(body.Wires()))
	}
}

// TestImprintBodiesSplitsContact: two stacked boxes imprint each other — the
// big box's top face splits along the small box's footprint, both bodies keep
// their volumes, and the touched faces/edges are reported.
func TestImprintBodiesSplitsContact(t *testing.T) {
	t.Parallel()
	big, _ := brep.SolidBlock(math.P3(0, 0, 0), math.P3(4, 4, 1), "big")
	small, _ := brep.SolidBlock(math.P3(1, 1, 1), math.P3(3, 3, 2), "small")
	ra, rb, err := brep.ImprintBodies(big, small)
	if err != nil {
		t.Fatalf("ImprintBodies: %v", err)
	}
	if v := vol(ra.Body); stdmath.Abs(v-16) > 1e-6 {
		t.Errorf("imprinted big volume = %g, want 16", v)
	}
	if v := vol(rb.Body); stdmath.Abs(v-4) > 1e-6 {
		t.Errorf("imprinted small volume = %g, want 4", v)
	}
	if len(ra.Body.Faces()) <= 6 {
		t.Errorf("big body has %d faces; the top should have split (>6)", len(ra.Body.Faces()))
	}
	if len(ra.TouchedFaces) == 0 || len(ra.ImprintedEdge) == 0 {
		t.Errorf("imprint should report touched faces (%d) and imprinted edges (%d)",
			len(ra.TouchedFaces), len(ra.ImprintedEdge))
	}
}
