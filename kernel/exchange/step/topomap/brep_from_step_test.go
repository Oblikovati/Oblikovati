// SPDX-License-Identifier: GPL-2.0-only

package topomap

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/exchange/step/geommap"
	"oblikovati/kernel/exchange/step/part21"
	"oblikovati/kernel/geom"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

// tetraShell is a minimal CLOSED_SHELL: a tetrahedron (4 triangular planar faces,
// 4 vertices, 6 edges). It is the smallest valid solid, exercising the full sense
// composition (EDGE_CURVE/ORIENTED_EDGE/ADVANCED_FACE) without curved geometry.
const tetraShell = `ISO-10303-21;
HEADER;
FILE_DESCRIPTION((''),'');
FILE_NAME('','',(''),(''),'','','');
FILE_SCHEMA(('CONFIG_CONTROL_DESIGN'));
ENDSEC;
DATA;
#1=CARTESIAN_POINT('',(0.,0.,0.));
#2=CARTESIAN_POINT('',(1.,0.,0.));
#3=CARTESIAN_POINT('',(0.,1.,0.));
#4=CARTESIAN_POINT('',(0.,0.,1.));
#11=VERTEX_POINT('',#1);
#12=VERTEX_POINT('',#2);
#13=VERTEX_POINT('',#3);
#14=VERTEX_POINT('',#4);
#21=DIRECTION('',(1.,0.,0.));
#22=DIRECTION('',(0.,1.,0.));
#23=DIRECTION('',(0.,0.,1.));
#31=VECTOR('',#21,1.);
#32=VECTOR('',#22,1.);
#33=VECTOR('',#23,1.);
#41=LINE('',#1,#31);
#42=LINE('',#1,#32);
#43=LINE('',#1,#33);
#44=LINE('',#2,#34);
#34=VECTOR('',#52,1.);
#52=DIRECTION('',(-1.,1.,0.));
#45=LINE('',#2,#35);
#35=VECTOR('',#53,1.);
#53=DIRECTION('',(-1.,0.,1.));
#46=LINE('',#3,#36);
#36=VECTOR('',#54,1.);
#54=DIRECTION('',(0.,-1.,1.));
#61=EDGE_CURVE('',#11,#12,#41,.T.);
#62=EDGE_CURVE('',#11,#13,#42,.T.);
#63=EDGE_CURVE('',#11,#14,#43,.T.);
#64=EDGE_CURVE('',#12,#13,#44,.T.);
#65=EDGE_CURVE('',#12,#14,#45,.T.);
#66=EDGE_CURVE('',#13,#14,#46,.T.);
#101=CARTESIAN_POINT('',(0.,0.,0.));
#102=AXIS2_PLACEMENT_3D('',#101,#23,#21);
#103=PLANE('',#102);
#104=ORIENTED_EDGE('',*,*,#61,.F.);
#105=ORIENTED_EDGE('',*,*,#62,.T.);
#106=ORIENTED_EDGE('',*,*,#64,.F.);
#107=EDGE_LOOP('',(#104,#106,#105));
#108=FACE_OUTER_BOUND('',#107,.T.);
#109=ADVANCED_FACE('',(#108),#103,.T.);
#111=AXIS2_PLACEMENT_3D('',#101,#22,#23);
#112=PLANE('',#111);
#113=ORIENTED_EDGE('',*,*,#61,.T.);
#114=ORIENTED_EDGE('',*,*,#65,.T.);
#115=ORIENTED_EDGE('',*,*,#63,.F.);
#116=EDGE_LOOP('',(#113,#114,#115));
#117=FACE_OUTER_BOUND('',#116,.T.);
#118=ADVANCED_FACE('',(#117),#112,.T.);
#121=AXIS2_PLACEMENT_3D('',#101,#21,#22);
#122=PLANE('',#121);
#123=ORIENTED_EDGE('',*,*,#62,.T.);
#124=ORIENTED_EDGE('',*,*,#66,.T.);
#125=ORIENTED_EDGE('',*,*,#63,.F.);
#126=EDGE_LOOP('',(#125,#123,#124));
#127=FACE_OUTER_BOUND('',#126,.T.);
#128=ADVANCED_FACE('',(#127),#122,.T.);
#131=CARTESIAN_POINT('',(1.,0.,0.));
#132=DIRECTION('',(0.5773502691896258,0.5773502691896258,0.5773502691896258));
#133=DIRECTION('',(-1.,1.,0.));
#134=AXIS2_PLACEMENT_3D('',#131,#132,#133);
#135=PLANE('',#134);
#136=ORIENTED_EDGE('',*,*,#64,.T.);
#137=ORIENTED_EDGE('',*,*,#66,.T.);
#138=ORIENTED_EDGE('',*,*,#65,.F.);
#139=EDGE_LOOP('',(#136,#137,#138));
#140=FACE_OUTER_BOUND('',#139,.T.);
#141=ADVANCED_FACE('',(#140),#135,.T.);
#200=CLOSED_SHELL('',(#109,#118,#128,#141));
ENDSEC;
END-ISO-10303-21;
`

func TestSolidFromShellSharesEdges(t *testing.T) {
	f, err := part21.Parse([]byte(tetraShell))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	body, warns, err := SolidFromShell(f.Graph, 200, true, 1.0, "test")
	if err != nil {
		t.Fatalf("SolidFromShell: %v", err)
	}
	if len(warns) != 0 {
		t.Errorf("unexpected warnings: %v", warns)
	}
	if got := len(body.Faces()); got != 4 {
		t.Errorf("tetra has %d faces, want 4", got)
	}
	if got := len(body.Edges()); got != 6 {
		t.Errorf("tetra has %d shared edges, want 6", got)
	}
}

func TestTopologyMalformedRecordsError(t *testing.T) {
	g := graphOfTopomap(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));\n"+
		"#2=DIRECTION('',(0.,0.,1.));\n"+
		"#3=VERTEX_POINT('',#1);\n"+
		"#4=EDGE_CURVE('',#3,#3,#2,.T.);\n"+
		"#5=CLOSED_SHELL('');\n"+
		"#6=EDGE_LOOP('',());\n"+
		"#7=FACE_OUTER_BOUND('',#6,.T.);")
	if ent, _ := g.Lookup(1); func() bool { _, err := shellFaceRefs(ent); return err == nil }() {
		t.Fatal("shellFaceRefs accepted a non-shell entity")
	}
	if ent, _ := g.Lookup(5); func() bool { _, err := shellFaceRefs(ent); return err == nil }() {
		t.Fatal("shellFaceRefs accepted a shell missing a face list")
	}
	a := newAssembler(g, 1.0, "test", false)
	if _, err := a.buildVertex(1); err == nil {
		t.Fatal("buildVertex accepted a CARTESIAN_POINT as VERTEX_POINT")
	}
	if _, err := a.readEdgeCurve(3); err == nil {
		t.Fatal("readEdgeCurve accepted a VERTEX_POINT")
	}
	if _, err := parseEdgeCurve(&part21.RawEntity{ID: 9, Keyword: "EDGE_CURVE", Params: nil}); err == nil {
		t.Fatal("parseEdgeCurve accepted too few params")
	}
	if _, err := a.buildBound(6); err == nil {
		t.Fatal("buildBound accepted EDGE_LOOP as FACE_BOUND")
	}
	if _, err := a.buildLoopUses(7, false); err == nil {
		t.Fatal("buildLoopUses accepted FACE_OUTER_BOUND as EDGE_LOOP")
	}
	if _, err := refParam(nil, 0); err == nil {
		t.Fatal("refParam accepted missing parameter")
	}
}

func TestCircleEdgeTrimsArcAndFullCircle(t *testing.T) {
	circle := geommap.CircleParams{
		Center: math.P3(0, 0, 0),
		Normal: math.V3(0, 0, 1),
		RefDir: math.V3(1, 0, 0),
		Radius: 2,
	}
	start := math.P3(2, 0, 0)
	end := math.P3(0, 2, 0)
	curve, err := circleEdge(circle, start, end, true)
	if err != nil {
		t.Fatalf("circleEdge arc: %v", err)
	}
	arc, ok := curve.(geom.Arc3d)
	if !ok {
		t.Fatalf("circleEdge returned %T, want Arc3d", curve)
	}
	if stdmath.Abs(arc.SweepAngle-stdmath.Pi/2) > 1e-12 {
		t.Fatalf("CCW sweep = %g, want pi/2", arc.SweepAngle)
	}
	curve, err = circleEdge(circle, start, end, false)
	if err != nil {
		t.Fatalf("circleEdge reversed arc: %v", err)
	}
	arc, ok = curve.(geom.Arc3d)
	if !ok {
		t.Fatalf("reversed circleEdge returned %T, want Arc3d", curve)
	}
	if stdmath.Abs(arc.SweepAngle+3*stdmath.Pi/2) > 1e-12 {
		t.Fatalf("CW sweep = %g, want -3*pi/2", arc.SweepAngle)
	}
	// A full-circle seam parametrizes FROM the seam vertex: PointAt(0) must equal the vertex, so
	// discretizeEdge's endpoint snap is a no-op and the boundary ring is a clean simple ring (not
	// the self-touching ring that produced a stray wedge across a bordering hole). Use a seam
	// vertex NOT at the circle's natural RefDir (+X) to exercise the alignment.
	seam := math.P3(0, 2, 0)
	curve, err = circleEdge(circle, seam, seam, true)
	if err != nil {
		t.Fatalf("circleEdge full circle: %v", err)
	}
	full, ok := curve.(geom.Arc3d)
	if !ok {
		t.Fatalf("full circle returned %T, want Arc3d", curve)
	}
	if stdmath.Abs(full.SweepAngle-2*stdmath.Pi) > 1e-12 {
		t.Fatalf("full-circle sweep = %g, want 2*pi", full.SweepAngle)
	}
	if p := full.PointAt(0); p.DistanceTo(seam) > 1e-9 {
		t.Fatalf("full circle PointAt(0) = %v, want the seam vertex %v", p, seam)
	}
}

func TestReverseUsesFlipsOrder(t *testing.T) {
	uses := []topo.Use{{Reversed: false}, {Reversed: true}}
	reverseUses(uses)
	if !uses[0].Reversed || uses[1].Reversed {
		t.Fatalf("reverseUses order = %+v", uses)
	}
}

func TestUnsupportedSurfaceWarnsAndSkipsFace(t *testing.T) {
	g := graphOfTopomap(t, "#1=SURFACE_OF_REVOLUTION('',*,*);\n"+
		"#2=ADVANCED_FACE('',(),#1,.T.);\n"+
		"#3=OPEN_SHELL('',(#2));")
	body, warns, err := SolidFromShell(g, 3, false, 1.0, "test")
	if err != nil {
		t.Fatalf("SolidFromShell unsupported surface: %v", err)
	}
	if len(warns) != 1 || warns[0] != "skipped face #2: geommap: unsupported surface SURFACE_OF_REVOLUTION (#1)" {
		t.Fatalf("warnings = %v", warns)
	}
	if got := len(body.Faces()); got != 0 {
		t.Fatalf("skipped body faces = %d, want 0", got)
	}
}

func graphOfTopomap(t *testing.T, stmts string) *part21.EntityGraph {
	t.Helper()
	src := "ISO-10303-21;\nHEADER;\nFILE_DESCRIPTION((''),'');\n" +
		"FILE_NAME('','',(''),(''),'','','');\nFILE_SCHEMA(('CONFIG_CONTROL_DESIGN'));\n" +
		"ENDSEC;\nDATA;\n" + stmts + "\nENDSEC;\nEND-ISO-10303-21;\n"
	f, err := part21.Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return f.Graph
}

func TestBodyToStepEmitsSharedTopology(t *testing.T) {
	f, err := part21.Parse([]byte(tetraShell))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	body, warns, err := SolidFromShell(f.Graph, 200, true, 1.0, "test")
	if err != nil {
		t.Fatalf("SolidFromShell: %v", err)
	}
	if len(warns) != 0 {
		t.Fatalf("unexpected warnings: %v", warns)
	}
	w := part21.NewWriter()
	emit := geommap.NewEmitter(w, 1.0)
	id, err := BodyToStep(emit, body)
	if err != nil {
		t.Fatalf("BodyToStep solid: %v", err)
	}
	out, err := part21.Parse(w.Emit(part21.Header{SchemaIdentifiers: []string{"CONFIG_CONTROL_DESIGN"}}))
	if err != nil {
		t.Fatalf("parse emitted solid: %v", err)
	}
	ent, err := out.Graph.Lookup(id)
	if err != nil {
		t.Fatalf("lookup emitted brep: %v", err)
	}
	if ent.Keyword != "MANIFOLD_SOLID_BREP" {
		t.Fatalf("emitted root = %s, want MANIFOLD_SOLID_BREP", ent.Keyword)
	}
	if got := len(out.Graph.EntitiesOfType("EDGE_CURVE")); got != 6 {
		t.Fatalf("EDGE_CURVE count = %d, want shared tetra edges 6", got)
	}
	if got := len(out.Graph.EntitiesOfType("ADVANCED_FACE")); got != 4 {
		t.Fatalf("ADVANCED_FACE count = %d, want 4", got)
	}
}

func TestBodyToStepEmitsSurfaceModelForOpenBody(t *testing.T) {
	f, err := part21.Parse([]byte(tetraShell))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	body, warns, err := SolidFromShell(f.Graph, 200, false, 1.0, "test")
	if err != nil {
		t.Fatalf("SolidFromShell surface: %v", err)
	}
	if len(warns) != 0 {
		t.Fatalf("unexpected warnings: %v", warns)
	}
	w := part21.NewWriter()
	emit := geommap.NewEmitter(w, 1.0)
	id, err := BodyToStep(emit, body)
	if err != nil {
		t.Fatalf("BodyToStep surface: %v", err)
	}
	out, err := part21.Parse(w.Emit(part21.Header{SchemaIdentifiers: []string{"CONFIG_CONTROL_DESIGN"}}))
	if err != nil {
		t.Fatalf("parse emitted surface: %v", err)
	}
	ent, err := out.Graph.Lookup(id)
	if err != nil {
		t.Fatalf("lookup emitted surface root: %v", err)
	}
	if ent.Keyword != "SHELL_BASED_SURFACE_MODEL" {
		t.Fatalf("emitted root = %s, want SHELL_BASED_SURFACE_MODEL", ent.Keyword)
	}
	if got := len(out.Graph.EntitiesOfType("OPEN_SHELL")); got != 1 {
		t.Fatalf("OPEN_SHELL count = %d, want 1", got)
	}
}
