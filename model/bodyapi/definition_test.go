// SPDX-License-Identifier: GPL-2.0-only

package bodyapi

import (
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
)

// Decoding the wire definition graph into the kernel graph. A geometric problem is REPORTED BY GRAPH
// PATH and rejects the whole graph — the decoder never returns a half-built definition.

// squareDef is a one-face definition: four vertices, four line-segment edges, one planar face in a
// single shell — the smallest complete graph the decoder accepts.
func squareDef() types.BrepBodyDefinition {
	corners := [][]float64{{0, 0, 0}, {2, 0, 0}, {2, 2, 0}, {0, 2, 0}}
	def := types.BrepBodyDefinition{Solid: false}
	for _, c := range corners {
		def.Vertices = append(def.Vertices, types.BrepVertexDef{Position: c})
	}
	var uses []types.BrepEdgeUseDef
	for i := range corners {
		j := (i + 1) % len(corners)
		def.Edges = append(def.Edges, types.BrepEdgeDef{
			Curve:       types.BrepCurveDef{Kind: "lineSegment", Points: append(append([]float64{}, corners[i]...), corners[j]...)},
			StartVertex: i, EndVertex: j,
		})
		uses = append(uses, types.BrepEdgeUseDef{Edge: i})
	}
	def.Faces = []types.BrepFaceDef{{
		Surface: types.BrepSurfaceDef{Kind: "plane", Origin: []float64{0, 0, 0}, Normal: []float64{0, 0, 1}},
		Loops:   []types.BrepLoopDef{{Uses: uses}},
	}}
	def.Lumps = []types.BrepLumpDef{{
		Shells: []types.BrepShellDef{{Faces: []int{0}}},
		Wires:  []types.BrepWireDef{{Edges: []int{0}}},
	}}
	return def
}

// TestDecodeBodyDefinitionKeepsTheGraph: a complete graph decodes with no issue and every part of it
// reaches the kernel definition.
func TestDecodeBodyDefinitionKeepsTheGraph(t *testing.T) {
	out, issues := DecodeBodyDefinition(squareDef())
	if len(issues) != 0 {
		t.Fatalf("valid definition reported issues: %v", issues)
	}
	if len(out.Vertices) != 4 || len(out.Edges) != 4 || len(out.Faces) != 1 {
		t.Fatalf("decoded %d vertices / %d edges / %d faces, want 4 / 4 / 1",
			len(out.Vertices), len(out.Edges), len(out.Faces))
	}
	if _, isPlane := out.Faces[0].Surface.(geom.Plane); !isPlane {
		t.Errorf("face surface is %T, want geom.Plane", out.Faces[0].Surface)
	}
	if len(out.Faces[0].Loops) != 1 || len(out.Faces[0].Loops[0].Uses) != 4 {
		t.Errorf("face loop decoded %d loops, want one loop of 4 uses", len(out.Faces[0].Loops))
	}
	if len(out.Lumps) != 1 || len(out.Lumps[0].Shells) != 1 || len(out.Lumps[0].Wires) != 1 {
		t.Errorf("lumps decoded as %+v, want one lump with one shell and one wire", out.Lumps)
	}
}

// TestDecodeBodyDefinitionRejectsWholeGraphOnIssue: one bad vertex, one bad curve and one bad surface
// each report their own path, and NO kernel definition comes back.
func TestDecodeBodyDefinitionRejectsWholeGraphOnIssue(t *testing.T) {
	def := squareDef()
	def.Vertices[1].Position = []float64{1, 2} // not a triplet
	def.Edges[2].Curve.Kind = "spiral"
	def.Faces[0].Surface.Kind = "hyperboloid"
	out, issues := DecodeBodyDefinition(def)
	if len(out.Vertices) != 0 || len(out.Faces) != 0 {
		t.Error("a rejected graph still returned a kernel definition")
	}
	paths := map[string]string{}
	for _, is := range issues {
		paths[is.Path] = is.Problem
	}
	for _, want := range []string{"vertices[1]", "edges[2]", "faces[0]"} {
		if _, ok := paths[want]; !ok {
			t.Errorf("no issue reported at %s (got %v)", want, paths)
		}
	}
	if !strings.Contains(paths["edges[2]"], "spiral") || !strings.Contains(paths["faces[0]"], "hyperboloid") {
		t.Errorf("issues %v do not name the offending kinds", paths)
	}
}

// TestDecodeCurveKinds covers every curve kind and its malformed form; each message names what the
// kind needs (the CLAUDE.md exception-message contract).
func TestDecodeCurveKinds(t *testing.T) {
	line := []float64{0, 0, 0, 1, 0, 0}
	cases := []struct {
		name string
		def  types.BrepCurveDef
		want string // "" = must decode
	}{
		{"lineSegment", types.BrepCurveDef{Kind: "lineSegment", Points: line}, ""},
		{"lineSegment short", types.BrepCurveDef{Kind: "lineSegment", Points: []float64{0, 0, 0}}, "2 points"},
		{"polyline", types.BrepCurveDef{Kind: "polyline", Points: []float64{0, 0, 0, 1, 0, 0, 2, 1, 0}}, ""},
		{"polyline short", types.BrepCurveDef{Kind: "polyline", Points: []float64{0, 0, 0}}, "2+ xyz triplets"},
		{"arc", types.BrepCurveDef{Kind: "arc", Center: []float64{0, 0, 0}, Normal: []float64{0, 0, 1},
			RefDir: []float64{1, 0, 0}, Radius: 2, SweepAngle: 1}, ""},
		{"arc no normal", types.BrepCurveDef{Kind: "arc", Center: []float64{0, 0, 0}, RefDir: []float64{1, 0, 0}}, "center, normal and refDir"},
		{"unknown", types.BrepCurveDef{Kind: "helix"}, "unknown curve kind"},
	}
	for _, c := range cases {
		_, err := decodeCurve(c.def)
		assertDecodeErr(t, c.name, err, c.want)
	}
}

// TestDecodeSurfaceKinds covers every surface kind and its malformed form.
func TestDecodeSurfaceKinds(t *testing.T) {
	o, ax := []float64{0, 0, 0}, []float64{0, 0, 1}
	cases := []struct {
		name string
		def  types.BrepSurfaceDef
		want string
	}{
		{"plane", types.BrepSurfaceDef{Kind: "plane", Origin: o, Normal: ax}, ""},
		{"plane no normal", types.BrepSurfaceDef{Kind: "plane", Origin: o}, "origin and normal"},
		{"cylinder", types.BrepSurfaceDef{Kind: "cylinder", Origin: o, Axis: ax, Radius: 2}, ""},
		{"cylinder no axis", types.BrepSurfaceDef{Kind: "cylinder", Origin: o, Radius: 2}, "origin and axis"},
		{"cone", types.BrepSurfaceDef{Kind: "cone", Origin: o, Axis: ax, HalfAngle: 0.5}, ""},
		{"sphere", types.BrepSurfaceDef{Kind: "sphere", Origin: o, Radius: 2}, ""},
		{"sphere no origin", types.BrepSurfaceDef{Kind: "sphere", Radius: 2}, "origin"},
		{"torus", types.BrepSurfaceDef{Kind: "torus", Origin: o, Axis: ax, MajorRadius: 4, MinorRadius: 1}, ""},
		{"unknown", types.BrepSurfaceDef{Kind: "hyperboloid"}, "unknown surface kind"},
	}
	for _, c := range cases {
		_, err := decodeSurface(c.def)
		assertDecodeErr(t, c.name, err, c.want)
	}
}

// assertDecodeErr checks a decode either succeeded (want == "") or failed with a message naming want.
func assertDecodeErr(t *testing.T, name string, err error, want string) {
	t.Helper()
	if want == "" {
		if err != nil {
			t.Errorf("%s: %v, want a decoded value", name, err)
		}
		return
	}
	if err == nil {
		t.Errorf("%s: decoded, want a decline naming %q", name, want)
		return
	}
	if !strings.Contains(err.Error(), want) {
		t.Errorf("%s: %v, want a message naming %q", name, err, want)
	}
}
