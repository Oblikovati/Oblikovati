// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/model/compdef"
)

// onePagePDF assembles a one-page, uncompressed PDF around a content-stream body — enough
// to exercise the model-level import without committing a binary fixture.
func onePagePDF(content string) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")
	b.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	b.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	b.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 720 720] /Contents 4 0 R >>\nendobj\n")
	fmt.Fprintf(&b, "4 0 obj\n<< >>\nstream\n%s\nendstream\nendobj\n", content)
	b.WriteString("trailer\n<< /Root 1 0 R >>\n")
	return b.Bytes()
}

// TestImportPDFPlanarOneSketch imports a vector PDF and checks it lands as a single 2D
// sketch on the chosen plane with the expected curve entities.
func TestImportPDFPlanarOneSketch(t *testing.T) {
	t.Parallel()
	part := compdef.NewPartComponentDefinition()
	content := "0 0 m 72 0 l 72 72 l S\n100 100 100 172 172 172 c S\n"
	res, err := ImportPDF(part, onePagePDF(content), xyPlane(t))
	if err != nil {
		t.Fatalf("ImportPDF: %v", err)
	}
	if res.Is3D {
		t.Errorf("planar PDF imported as 3D")
	}
	if part.Sketches().Count() != 1 || part.Sketches3D().Count() != 0 {
		t.Fatalf("expected one 2D sketch, got %d/%d", part.Sketches().Count(), part.Sketches3D().Count())
	}
	sk := part.Sketches().Item(0)
	if got := sk.Lines().Count(); got != 2 { // the 3-point stroke is two line segments
		t.Errorf("lines = %d, want 2", got)
	}
	if got := sk.Splines().Count(); got != 1 {
		t.Errorf("splines = %d, want 1", got)
	}
}

// TestImportPDFRealFile imports a real AutoCAD plot (corpus-gated) and checks the dense
// drawing lands as one 2D sketch.
func TestImportPDFRealFile(t *testing.T) {
	t.Parallel()
	dir := os.Getenv("PDF_TESTFILES_DIR")
	if dir == "" {
		dir = filepath.Join("..", "..", "..")
	}
	data, err := os.ReadFile(filepath.Join(dir, "pdf-vector2.pdf"))
	if err != nil {
		t.Skipf("corpus unavailable: %v", err)
	}
	part := compdef.NewPartComponentDefinition()
	res, err := ImportPDF(part, data, xyPlane(t))
	if err != nil {
		t.Fatalf("ImportPDF: %v", err)
	}
	if res.EntityCount < 100 {
		t.Errorf("imported only %d entities", res.EntityCount)
	}
	if part.Sketches().Count() != 1 {
		t.Errorf("expected one 2D sketch, got %d", part.Sketches().Count())
	}
}
