// SPDX-License-Identifier: GPL-2.0-only

package dxf

import (
	"testing"

	"oblikovati.org/kernel/exchange/drawing"
)

// lineCirclePointDXF is a minimal ASCII DXF (LF, unpadded codes) with a HEADER carrying
// $INSUNITS and an ENTITIES section with one LINE, CIRCLE and POINT.
const lineCirclePointDXF = `0
SECTION
2
HEADER
9
$INSUNITS
70
4
0
ENDSEC
0
SECTION
2
ENTITIES
0
LINE
5
2A
8
0
10
0.0
20
0.0
30
0.0
11
10.0
21
5.0
31
0.0
0
CIRCLE
5
2B
10
3.0
20
4.0
30
0.0
40
2.5
0
POINT
5
2C
10
7.0
20
8.0
30
0.0
0
ENDSEC
0
EOF
`

// TestDecodeLineCirclePoint checks the three simple entities decode with their coordinates,
// handles and the header $INSUNITS code.
func TestDecodeLineCirclePoint(t *testing.T) {
	dr, warns, err := Decode([]byte(lineCirclePointDXF))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(warns) != 0 {
		t.Errorf("unexpected warnings: %v", warns)
	}
	if dr.Units != 4 {
		t.Errorf("units = %d, want 4 (mm)", dr.Units)
	}
	if len(dr.Entities) != 3 {
		t.Fatalf("entities = %d, want 3", len(dr.Entities))
	}
	l, ok := dr.Entities[0].(*drawing.Line)
	if !ok || l.Start != [3]float64{0, 0, 0} || l.End != [3]float64{10, 5, 0} || l.Handle != 0x2A {
		t.Errorf("line = %+v", dr.Entities[0])
	}
	c, ok := dr.Entities[1].(*drawing.Circle)
	if !ok || c.Center != [3]float64{3, 4, 0} || c.Radius != 2.5 || c.Handle != 0x2B {
		t.Errorf("circle = %+v", dr.Entities[1])
	}
	if c.Normal != [3]float64{0, 0, 1} {
		t.Errorf("circle normal = %v, want +Z default", c.Normal)
	}
	p, ok := dr.Entities[2].(*drawing.Point)
	if !ok || p.Position != [3]float64{7, 8, 0} || p.Handle != 0x2C {
		t.Errorf("point = %+v", dr.Entities[2])
	}
}

// paddedCRLFLineDXF mirrors how AutoCAD/LibreDWG write files: codes padded to width 3,
// integer values right-justified, CRLF line endings. A robust reader must accept this.
const paddedCRLFLineDXF = "  0\r\nSECTION\r\n  2\r\nENTITIES\r\n  0\r\nLINE\r\n" +
	"  5\r\n34\r\n  8\r\n0\r\n 10\r\n1.5\r\n 20\r\n2.5\r\n 30\r\n0.0\r\n" +
	" 11\r\n4.0\r\n 21\r\n6.0\r\n 31\r\n0.0\r\n  0\r\nENDSEC\r\n  0\r\nEOF\r\n"

// TestDecodeToleratesPaddingAndCRLF checks the reader handles padded codes and CRLF.
func TestDecodeToleratesPaddingAndCRLF(t *testing.T) {
	dr, _, err := Decode([]byte(paddedCRLFLineDXF))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(dr.Entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(dr.Entities))
	}
	l := dr.Entities[0].(*drawing.Line)
	if l.Start != [3]float64{1.5, 2.5, 0} || l.End != [3]float64{4, 6, 0} {
		t.Errorf("line = %+v", l)
	}
}

// TestDecodeSkipsUnknownEntity checks an unsupported entity type is dropped without a
// warning or error, leaving the supported ones.
func TestDecodeSkipsUnknownEntity(t *testing.T) {
	const dxf = `0
SECTION
2
ENTITIES
0
TEXT
1
hello
10
0.0
20
0.0
0
LINE
10
0.0
20
0.0
11
1.0
21
1.0
0
ENDSEC
0
EOF
`
	dr, warns, err := Decode([]byte(dxf))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(warns) != 0 {
		t.Errorf("unexpected warnings: %v", warns)
	}
	if len(dr.Entities) != 1 {
		t.Fatalf("entities = %d, want 1 (TEXT skipped)", len(dr.Entities))
	}
	if dr.Entities[0].Kind() != drawing.KindLine {
		t.Errorf("kept %v, want LINE", dr.Entities[0].Kind())
	}
}

// TestDecodeUnitlessWhenNoHeader checks a file without $INSUNITS reports unitless.
func TestDecodeUnitlessWhenNoHeader(t *testing.T) {
	dr, _, err := Decode([]byte("0\nSECTION\n2\nENTITIES\n0\nENDSEC\n0\nEOF\n"))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if dr.Units != drawing.INSUnitless {
		t.Errorf("units = %d, want unitless", dr.Units)
	}
}

// TestScanPairsRejectsNonIntegerCode checks a malformed group code is a hard error.
func TestScanPairsRejectsNonIntegerCode(t *testing.T) {
	if _, err := scanPairs([]byte("notacode\nvalue\n")); err == nil {
		t.Error("expected error for non-integer group code")
	}
}

// TestDecodeBadCoordinateWarns checks a malformed coordinate is reported as a per-entity
// warning rather than aborting the decode.
func TestDecodeBadCoordinateWarns(t *testing.T) {
	const dxf = `0
SECTION
2
ENTITIES
0
LINE
10
notanumber
20
0.0
0
ENDSEC
0
EOF
`
	dr, warns, err := Decode([]byte(dxf))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(warns) != 1 {
		t.Fatalf("warnings = %v, want 1", warns)
	}
	if len(dr.Entities) != 0 {
		t.Errorf("entities = %d, want 0 (bad line dropped)", len(dr.Entities))
	}
}
