// SPDX-License-Identifier: GPL-2.0-only

package pdf

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"math"
	"testing"

	"oblikovati.org/kernel/exchange/drawing"
)

// minimalPDF assembles a one-page, uncompressed PDF around the given content-stream body.
// The decoder resolves objects by header scan and reads /Root from the trailer, so no xref
// table is needed — keeping the fixture readable.
func minimalPDF(content string) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")
	b.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	b.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	b.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 720 720] /Contents 4 0 R >>\nendobj\n")
	fmt.Fprintf(&b, "4 0 obj\n<< >>\nstream\n%s\nendstream\nendobj\n", content)
	b.WriteString("trailer\n<< /Root 1 0 R >>\n")
	return b.Bytes()
}

// approxMM compares a millimetre coordinate against a point value converted to millimetres.
func approxMM(t *testing.T, got, wantPoints float64) {
	t.Helper()
	want := wantPoints * mmPerPoint
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("coordinate = %g mm, want %g mm", got, want)
	}
}

// TestDecodeLineRectBezier covers the three path shapes a CAD plot uses: a stroked line
// (open polyline), a rectangle (closed polyline), and a cubic Bézier (degree-3 spline).
func TestDecodeLineRectBezier(t *testing.T) {
	content := "1 0 0 1 0 0 cm\n" +
		"0 0 m 72 0 l S\n" +
		"100 0 100 100 re S\n" +
		"200 200 m 200 272 272 272 272 200 c S\n"
	dr, warns, err := Decode(minimalPDF(content))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(warns) != 0 {
		t.Fatalf("unexpected warnings: %v", warns)
	}
	if dr.Units != drawing.INSMillimetres {
		t.Errorf("Units = %d, want %d (mm)", dr.Units, drawing.INSMillimetres)
	}
	polys, splines := classify(dr.Entities)
	if len(polys) != 2 || len(splines) != 1 {
		t.Fatalf("got %d polylines + %d splines, want 2 + 1", len(polys), len(splines))
	}
	assertLine(t, polys)
	assertRect(t, polys)
	assertBezier(t, splines[0])
}

// assertLine checks the open two-point polyline from "0 0 m 72 0 l S".
func assertLine(t *testing.T, polys []*drawing.LwPolyline) {
	t.Helper()
	line := findOpenPolyline(polys, 2)
	if line == nil {
		t.Fatal("no open 2-point polyline (the line) found")
	}
	approxMM(t, line.Points[1][0], 72)
	approxMM(t, line.Points[1][1], 0)
}

// assertRect checks the closed four-point polyline from "100 0 100 100 re".
func assertRect(t *testing.T, polys []*drawing.LwPolyline) {
	t.Helper()
	for _, p := range polys {
		if p.Closed && len(p.Points) == 4 {
			approxMM(t, p.Points[0][0], 100)
			approxMM(t, p.Points[2][1], 100)
			return
		}
	}
	t.Fatal("no closed 4-point polyline (the rectangle) found")
}

// assertBezier checks the degree-3, four-control-point spline from the c operator.
func assertBezier(t *testing.T, sp *drawing.Spline) {
	t.Helper()
	if sp.Degree != 3 || len(sp.ControlPoints) != 4 {
		t.Fatalf("spline degree=%d controls=%d, want 3 and 4", sp.Degree, len(sp.ControlPoints))
	}
	approxMM(t, sp.ControlPoints[0][0], 200) // start
	approxMM(t, sp.ControlPoints[3][0], 272) // end
}

// TestDecodeClipPathDiscarded asserts a W n clip path emits no geometry (page borders).
func TestDecodeClipPathDiscarded(t *testing.T) {
	dr, _, err := Decode(minimalPDF("0 0 m 100 0 l 100 100 l W n\n"))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(dr.Entities) != 0 {
		t.Errorf("clip path produced %d entities, want 0", len(dr.Entities))
	}
}

// TestDecodeCTMScalesCoordinates asserts the current transformation matrix is applied: a
// 2× scale doubles the device coordinate before the point→mm conversion.
func TestDecodeCTMScalesCoordinates(t *testing.T) {
	dr, _, err := Decode(minimalPDF("2 0 0 2 0 0 cm\n0 0 m 10 0 l S\n"))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	polys, _ := classify(dr.Entities)
	if len(polys) != 1 {
		t.Fatalf("got %d polylines, want 1", len(polys))
	}
	approxMM(t, polys[0].Points[1][0], 20) // 10 user units × 2 (CTM) = 20 points
}

// TestFlateDecodeContentStream covers the FlateDecode path: a compressed content stream
// decodes to the same geometry as its plaintext form.
func TestFlateDecodeContentStream(t *testing.T) {
	var raw bytes.Buffer
	w := zlib.NewWriter(&raw)
	if _, err := w.Write([]byte("0 0 m 72 0 l S\n")); err != nil {
		t.Fatal(err)
	}
	w.Close()
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")
	b.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	b.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	b.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 720 720] /Contents 4 0 R >>\nendobj\n")
	fmt.Fprintf(&b, "4 0 obj\n<< /Filter /FlateDecode /Length %d >>\nstream\n", raw.Len())
	b.Write(raw.Bytes())
	b.WriteString("\nendstream\nendobj\ntrailer\n<< /Root 1 0 R >>\n")
	dr, _, err := Decode(b.Bytes())
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	polys, _ := classify(dr.Entities)
	if len(polys) != 1 || len(polys[0].Points) != 2 {
		t.Fatalf("FlateDecode content gave %d polylines", len(polys))
	}
}

// TestDecodeRejectsObjectStreamPDF asserts an xref-stream / object-stream PDF (no classic
// indirect objects) fails with a clear, scoped error rather than importing nothing.
func TestDecodeRejectsObjectStreamPDF(t *testing.T) {
	_, _, err := Decode([]byte("%PDF-1.7\n%%EOF\n"))
	if err == nil {
		t.Fatal("expected an error for a PDF with no classic objects")
	}
}

// classify splits decoded entities into polylines and splines for assertions.
func classify(ents []drawing.Entity) ([]*drawing.LwPolyline, []*drawing.Spline) {
	var polys []*drawing.LwPolyline
	var splines []*drawing.Spline
	for _, e := range ents {
		switch g := e.(type) {
		case *drawing.LwPolyline:
			polys = append(polys, g)
		case *drawing.Spline:
			splines = append(splines, g)
		}
	}
	return polys, splines
}

// findOpenPolyline returns the first open polyline with n points.
func findOpenPolyline(polys []*drawing.LwPolyline, n int) *drawing.LwPolyline {
	for _, p := range polys {
		if !p.Closed && len(p.Points) == n {
			return p
		}
	}
	return nil
}
