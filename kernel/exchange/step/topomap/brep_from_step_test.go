// SPDX-License-Identifier: GPL-2.0-only

package topomap

import (
	"testing"

	"oblikovati/kernel/exchange/step/part21"
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
