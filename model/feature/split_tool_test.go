// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Splitting with a SURFACE tool rather than a work plane (#1891). The point of the tool types is
// that the cutting geometry comes from the model — a parting surface built by earlier features —
// instead of a datum the user has to add alongside it.

// boxAndSheetPart returns a part holding a 4×4×4 solid and a planar sheet across it at z=2.
func boxAndSheetPart() *PartFeatures {
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(subd.ToBody(subd.Box(4, 4, 4), "box"))
	NewBaseFeatures(fs).AddBase(patchSurfaceAtZ(4, 4, 2))
	return fs
}

// solidVolumes returns the volumes of the result's solid bodies, in order.
func solidVolumes(fs *PartFeatures) []float64 {
	var out []float64
	for _, b := range fs.Result() {
		if b.IsSolid() {
			out = append(out, ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume)
		}
	}
	return out
}

// TestSplitBySurfaceBodyHalvesTheSolid: the sheet at z=2 cuts the 4³ box into two 32s, and the
// sheet itself survives — it is the tool, not a target.
func TestSplitBySurfaceBodyHalvesTheSolid(t *testing.T) {
	t.Parallel()
	fs := boxAndSheetPart()
	sp := NewModifyFeatures(fs).AddSplitByDefinition(&SplitSolidDefinition{
		Tool: SplitBySurfaceBody, ToolIndex: 1, Keep: SplitBoth,
	})
	fs.Recompute()

	if !sp.Health().OK() {
		t.Fatalf("split by a surface body went sick: %+v", sp.Health())
	}
	vols := solidVolumes(fs)
	if len(vols) != 2 {
		t.Fatalf("split produced %d solids (%v), want 2", len(vols), vols)
	}
	for i, v := range vols {
		if stdmath.Abs(v-32) > 1e-6 {
			t.Errorf("piece %d volume = %g, want 32", i, v)
		}
	}
	if len(fs.Result()) != 3 {
		t.Errorf("result has %d bodies, want 3 — the cutting sheet must pass through", len(fs.Result()))
	}
}

// TestSplitByWorkSurfaceAddressesTheSheets: the work-surface tool indexes the part's SURFACES,
// not its bodies, which is a different numbering the moment the part holds a solid too — index 0
// here is the sheet, while body 0 is the box.
func TestSplitByWorkSurfaceAddressesTheSheets(t *testing.T) {
	t.Parallel()
	fs := boxAndSheetPart()
	sp := NewModifyFeatures(fs).AddSplitByDefinition(&SplitSolidDefinition{
		Tool: SplitByWorkSurface, ToolIndex: 0, Keep: SplitPositive,
	})
	fs.Recompute()

	if !sp.Health().OK() {
		t.Fatalf("split by a work surface went sick: %+v", sp.Health())
	}
	vols := solidVolumes(fs)
	if len(vols) != 1 || stdmath.Abs(vols[0]-32) > 1e-6 {
		t.Errorf("trim by the work surface left %v, want a single 32", vols)
	}
}

// bentSheetBody returns a two-face sheet whose halves lie in different planes — the simplest body
// that has no single cutting plane.
func bentSheetBody() *topo.Body {
	lin := topo.NewLineage(topo.Tok("test", "bent", 0))
	bld := topo.NewBuilder(false, lin)
	quad := func(pts [4]math.Point3, n math.Vector3) {
		v := make([]*topo.Vertex, 4)
		for i, p := range pts {
			v[i] = bld.AddVertex(p, lin)
		}
		uses := make([]topo.Use, 4)
		for i := range pts {
			uses[i] = topo.Use{Edge: bld.AddEdge(geom.NewLineSegment(pts[i], pts[(i+1)%4]), v[i], v[(i+1)%4], lin)}
		}
		pl, _ := geom.NewPlane(pts[0], n)
		bld.AddFace(pl, lin, topo.OuterLoop(uses...))
	}
	quad([4]math.Point3{math.P3(0, 0, 2), math.P3(2, 0, 2), math.P3(2, 4, 2), math.P3(0, 4, 2)}, math.V3(0, 0, 1))
	quad([4]math.Point3{math.P3(2, 0, 2), math.P3(4, 0, 4), math.P3(4, 4, 4), math.P3(2, 4, 2)}, math.V3(-1, 0, 1))
	return bld.Build()
}

// TestSplitRefusesANonPlanarTool: a bent or curved cutting sheet has no plane to extend, and
// approximating it by one of its faces would trim the part along the wrong surface. The refusal
// is the honest answer until there is a general surface partitioner.
func TestSplitRefusesANonPlanarTool(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(subd.ToBody(subd.Box(4, 4, 4), "box"))
	NewBaseFeatures(fs).AddBase(bentSheetBody())
	sp := NewModifyFeatures(fs).AddSplitByDefinition(&SplitSolidDefinition{Tool: SplitBySurfaceBody, ToolIndex: 1})
	fs.Recompute()

	if sp.Health().OK() {
		t.Fatal("splitting by a bent sheet should be refused, not approximated")
	}
}

// TestSplitToolMisuseIsRefused: a solid tool is a combine, an out-of-range index names nothing,
// and a path tool with no sketch has nothing to project. Each case asserts the REASON as well as the
// refusal — a solid tool would fail anyway for having no single plane, and "not planar" would
// send the caller looking for a flatter solid instead of the surface they meant to pick.
func TestSplitToolMisuseIsRefused(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		def  *SplitSolidDefinition
		want string
	}{
		"solid as tool":  {&SplitSolidDefinition{Tool: SplitBySurfaceBody, ToolIndex: 0}, "is a SOLID"},
		"out of range":   {&SplitSolidDefinition{Tool: SplitBySurfaceBody, ToolIndex: 9}, "out of range"},
		"no surfaces":    {&SplitSolidDefinition{Tool: SplitByWorkSurface, ToolIndex: 3}, "out of range"},
		"path no sketch": {&SplitSolidDefinition{Tool: SplitByPath}, "needs a sketch"},
		"no plane":       {&SplitSolidDefinition{Tool: SplitByWorkPlane}, "no cutting plane"},
	} {
		t.Run(name, func(t *testing.T) {
			fs := boxAndSheetPart()
			sp := NewModifyFeatures(fs).AddSplitByDefinition(c.def)
			fs.Recompute()
			if sp.Health().OK() {
				t.Fatalf("%s should make the split sick", name)
			}
			if !strings.Contains(sp.Health().Reason, c.want) {
				t.Errorf("%s reason = %q, want it to say %q", name, sp.Health().Reason, c.want)
			}
		})
	}
}

// TestSplitToolRoundTrips: the tool and its index must survive the recipe, and a work-plane split
// must keep writing its plane reference alone — an existing document is not touched by the tool
// option existing.
func TestSplitToolRoundTrips(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewModifyFeatures(fs).AddSplitByDefinition(&SplitSolidDefinition{
		Tool: SplitBySurfaceBody, ToolIndex: 2, Keep: SplitNegative,
	})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if d := data[0].SplitSolid; d.Tool != "surfaceBody" || d.ToolIndex != 2 || d.Plane != "" {
		t.Fatalf("serialized split = %+v, want the surfaceBody tool at 2 and no plane", d)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, NewWorkGeometry()); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	def := fresh.Item(0).Definition().(*SplitSolidFeature).Definition()
	if def.Tool != SplitBySurfaceBody || def.ToolIndex != 2 || def.Keep != SplitNegative {
		t.Errorf("restored split = %+v, want the surfaceBody tool at 2 keeping the negative side", def)
	}
}

// TestUnknownSplitToolNameIsRefused: a misspelled tool must not fall back to the work plane,
// which would cut along a datum the caller never asked for.
func TestUnknownSplitToolNameIsRefused(t *testing.T) {
	t.Parallel()
	if _, ok := ParseSplitTool("surface"); ok {
		t.Error(`ParseSplitTool("surface") should not resolve`)
	}
	if k, ok := ParseSplitTool(""); !ok || k != SplitByWorkPlane {
		t.Errorf("the empty tool name should be the work plane, got %v/%v", k, ok)
	}
	if n := SplitToolName(SplitByWorkPlane); n != "" {
		t.Errorf("the work-plane tool should serialize as nothing, got %q", n)
	}
}
