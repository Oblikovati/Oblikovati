// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"errors"
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// ringAndDrill is the RING corpus body and an axial drill of the given radius, placed on the ring's
// centreline circle exactly as the exact row (TestAxialDrillThroughARingIsExact, bore 0.8) places it.
// Shrinking only the radius is what makes it a size corpus: the configuration is identical, and the
// only thing that changes is whether the tool's material survives the model's seam weld.
func ringAndDrill(t *testing.T, bore float64) (*topo.Body, *topo.Body) {
	t.Helper()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	drill, err := brep.SolidCylinder(math.P3(5, 0, -4), math.V3(0, 0, 1), bore, 8)
	if err != nil {
		t.Fatalf("drill r=%g: %v", bore, err)
	}
	return ring, drill
}

// TestASubResolutionDrillIsRefusedByName is the corpus row for the size classification: the radii the
// sweep below shows the pipeline answered SILENTLY — the ring came back unchanged, err=nil, one face,
// removed=0, nothing recorded. Both sit inside the silent band (thickness <= 0.0998 x Weld).
func TestASubResolutionDrillIsRefusedByName(t *testing.T) {
	t.Parallel()
	for _, bore := range []float64{1e-10, 1e-9} {
		ring, drill := ringAndDrill(t, bore)
		rec := &diag.Recorder{}
		body, err := BooleanWithDiagnostics(Cut, ring, drill, rec)
		if !errors.Is(err, ErrSubResolutionOperand) {
			t.Fatalf("bore %g: want the named sub-resolution refusal; got err=%v", bore, err)
		}
		if body != nil {
			t.Fatalf("bore %g: a refused boolean must return no body; got %d faces", bore, len(body.Faces()))
		}
		if !rec.Has(CodeBooleanSubResolutionTool) {
			t.Errorf("bore %g: the refusal must reach the diagnostic channel; got %v", bore, rec.Records())
		}
	}
}

// The refusal has to name the offending value and what was expected of it (the CLAUDE.md rule): the
// user's remedy is to re-author at a working unit, which they cannot discover from a bare failure.
func TestTheSubResolutionRefusalNamesTheThicknessAndTheFloor(t *testing.T) {
	t.Parallel()
	ring, drill := ringAndDrill(t, 1e-10)
	rec := &diag.Recorder{}
	_, err := BooleanWithDiagnostics(Cut, ring, drill, rec)
	for _, want := range []string{"cut tool", "2e-10", "below this model's resolution", "working unit"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal must name %q; got %q", want, err.Error())
		}
	}
	if got := rec.Records(); len(got) != 1 || got[0].Severity != diag.Defect {
		t.Fatalf("want exactly one Defect on the recorder; got %v", got)
	}
}

// The classification runs BEFORE any geometry: nothing is intersected, imprinted or stitched, so the
// refusal carries no other diagnostic. A pipeline that had already run would have recorded its own.
func TestTheSubResolutionRefusalPrecedesTheGeometry(t *testing.T) {
	t.Parallel()
	ring, drill := ringAndDrill(t, 1e-10)
	rec := &diag.Recorder{}
	if _, err := BooleanWithDiagnostics(Cut, ring, drill, rec); err == nil {
		t.Fatal("want a refusal")
	}
	if rec.Has(CodeBooleanNoExactCurvedPath) || rec.Has(CodeBooleanAnalyticVolumeReject) {
		t.Errorf("the size classification must refuse before the pipeline runs; got %v", rec.Records())
	}
}

// The public curved entry is the same operation and had the same hole: CurvedBoolean certified the
// unchanged ring as a Cut result — a valid body, every face accounted for, and a volume the Requicha
// bracket admits, because "removed nothing" is inside [V(A)−V(B), V(A)].
func TestTheCurvedEntryAlsoRefusesASubResolutionTool(t *testing.T) {
	t.Parallel()
	ring, drill := ringAndDrill(t, 1e-10)
	rec := &diag.Recorder{}
	body, ok := CurvedBooleanWithDiagnostics(Cut, ring, drill, rec)
	if ok {
		v := query.BodyGeometryProperties(body, DefaultQuality()).Volume
		t.Fatalf("the curved entry accepted a sub-resolution cut: volume %g (the ring is 222.0661)", v)
	}
	if !rec.Has(CodeBooleanSubResolutionTool) {
		t.Errorf("the curved entry's refusal must be named too; got %v", rec.Records())
	}
}

// The floor is the model's SEAM weld, not an absolute size: the same drill is refused in a big model
// and built in a small one, because what fails is the ratio. A 1e-6 bore through a ring a thousand
// times smaller is ordinary geometry.
func TestTheSubResolutionFloorIsModelRelative(t *testing.T) {
	t.Parallel()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5e-3, 1.5e-3, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	drill, err := brep.SolidCylinder(math.P3(5e-3, 0, -4e-3), math.V3(0, 0, 1), 8e-4, 8e-3)
	if err != nil {
		t.Fatalf("drill: %v", err)
	}
	if err := declineSubResolutionOperand(Cut, ring, drill, nil); err != nil {
		t.Fatalf("a millimetre-scale ring and its proportional drill must classify as modellable: %v", err)
	}
}

// The boolean's floor and the UI's feature-scale warning must be ONE predicate: a size the UI calls
// resolvable that the boolean then refuses is the disagreement finding 3 of the stage-6 review
// found (they answered at 1e-9 and 1e-6 of model size). This drives both through the same drill.
func TestTheBooleanFloorAgreesWithTheFeatureScaleWarning(t *testing.T) {
	t.Parallel()
	for _, bore := range []float64{1e-10, 1e-9, 1e-8, 1e-6, 1e-3, 0.8} {
		ring, drill := ringAndDrill(t, bore)
		box := ring.RangeBox().Union(drill.RangeBox())
		thickness, ok := solidThickness(drill)
		if !ok {
			t.Fatalf("bore %g: the drill is a solid and must measure", bore)
		}
		uiSaysResolvable := geom.FeatureResolvable(box, thickness)
		booleanRefuses := declineSubResolutionOperand(Cut, ring, drill, nil) != nil
		if uiSaysResolvable == booleanRefuses {
			t.Errorf("bore %g (thickness %g): the UI says resolvable=%v while the boolean refuses=%v — "+
				"the two policies must be one predicate", bore, thickness, uiSaysResolvable, booleanRefuses)
		}
		if warn := geom.SpanCeilingWarning(box, thickness); (warn != "") != booleanRefuses {
			t.Errorf("bore %g: SpanCeilingWarning=%q disagrees with the boolean refusal=%v", bore, warn, booleanRefuses)
		}
	}
}

// The remedy sentence has ONE home. It used to be typed out twice — once in geom.SpanCeilingWarning
// and once in the boolean's decline — which is two things to keep in step and two things for the
// user to read as if they were different advice.
func TestBothRefusalsCarryTheOneRemedySentence(t *testing.T) {
	t.Parallel()
	ring, drill := ringAndDrill(t, 1e-10)
	thickness, _ := solidThickness(drill)
	err := declineSubResolutionOperand(Cut, ring, drill, nil)
	if err == nil {
		t.Fatal("want a refusal")
	}
	warn := geom.SpanCeilingWarning(ring.RangeBox().Union(drill.RangeBox()), thickness)
	for name, text := range map[string]string{"the boolean refusal": err.Error(), "SpanCeilingWarning": warn} {
		if !strings.Contains(text, geom.ScaleRemedy) {
			t.Errorf("%s must carry geom.ScaleRemedy verbatim; got %q", name, text)
		}
	}
}

// The exact row must stay exact: the classification is a floor, not a new gate on ordinary geometry.
func TestTheExactDrillRowStillPassesTheSizeClassification(t *testing.T) {
	t.Parallel()
	ring, drill := ringAndDrill(t, 0.8)
	if err := declineSubResolutionOperand(Cut, ring, drill, nil); err != nil {
		t.Fatalf("the RD− corpus row must not be refused on size: %v", err)
	}
}

// squareSheet is a named fixture: one planar face declared NON-solid, the shape a split/replace-face
// feature cuts with. Its bounding box is flat, so a thickness test that ignored IsSolid would refuse it.
func squareSheet(t *testing.T) *topo.Body {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "sheet", 0))
	bld := topo.NewBuilder(false, lin)
	corners := []math.Point3{math.P3(0, 0, 0), math.P3(4, 0, 0), math.P3(4, 4, 0), math.P3(0, 4, 0)}
	uses := make([]topo.Use, len(corners))
	verts := make([]*topo.Vertex, len(corners))
	for i, p := range corners {
		verts[i] = bld.AddVertex(p, lin)
	}
	for i := range corners {
		j := (i + 1) % len(corners)
		uses[i] = topo.Fwd(bld.AddEdge(geom.NewLineSegment(corners[i], corners[j]), verts[i], verts[j], lin))
	}
	plane, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	bld.AddFace(plane, lin, topo.OuterLoop(uses...))
	return bld.Build()
}

// solidThickness measures MATERIAL, so it declines to measure what has none: a sheet body's zero
// extent is its representation, not a part too thin to build, and refusing every planar surface tool
// would break the split/replace-face features that cut with one.
func TestSolidThicknessDeclinesASheetBody(t *testing.T) {
	t.Parallel()
	sheet := squareSheet(t)
	if _, ok := solidThickness(sheet); ok {
		t.Error("a sheet body has no material thickness to measure")
	}
	if err := declineSubResolutionOperand(Cut, sheet, sheet, nil); err != nil {
		t.Errorf("a sheet operand must not be refused on thickness: %v", err)
	}
}

// TestSolidThicknessIsTheSmallestExtent pins the measure itself: the thinnest direction of the
// material, not the diagonal (which a long thin drill passes) and not the largest side.
func TestSolidThicknessIsTheSmallestExtent(t *testing.T) {
	t.Parallel()
	slab, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(10, 4, 0.25), "slab")
	if err != nil {
		t.Fatalf("slab: %v", err)
	}
	got, ok := solidThickness(slab)
	if !ok || stdmath.Abs(got-0.25) > 1e-12 { // tol:numeric — an exact box extent, float noise only
		t.Errorf("solidThickness(10x4x0.25 block) = %g, %v; want 0.25", got, ok)
	}
	if _, ok := solidThickness(nil); ok {
		t.Error("a nil body has no thickness")
	}
}
