// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "testdata", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return b
}

// TestOpenPartClassifies verifies a real CFBF .SLDPRT opens, is classified as a Part by its
// root CLSID, and carries the SolidWorks build stamp (3400) parsed from _MO_VERSION_3400.
func TestOpenPartClassifies(t *testing.T) {
	d, err := Open(readTestdata(t, "stud_cfbf.sldprt"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if d.Type != Part {
		t.Errorf("Type = %v, want part", d.Type)
	}
	if d.Version != 3400 {
		t.Errorf("Version = %d, want 3400", d.Version)
	}
	// The feature tree + geometry live in these streams; they must be present.
	for _, s := range []string{"Contents/Config-0", "Contents/Config-0-Partition", "Contents/Definition"} {
		if _, err := d.Stream(s); err != nil {
			t.Errorf("missing stream %q: %v", s, err)
		}
	}
}

// TestDocTypeString covers the DocType stringer.
func TestDocTypeString(t *testing.T) {
	cases := map[DocType]string{Part: "part", Assembly: "assembly", Drawing: "drawing", Unknown: "unknown"}
	for dt, want := range cases {
		if got := dt.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", dt, got, want)
		}
	}
}

// TestOpenFormatB verifies the SolidWorks-2026 non-OLE container (format B) decodes through the
// same interface as CFBF: Open parses the record log, and Config-0 inflates to the SAME MFC
// CArchive object graph as format A (its header is the `moPart_c` class registration), so both
// formats share one downstream decoder.
func TestOpenFormatB(t *testing.T) {
	d, err := Open(readTestdata(t, "perno_fmtb.sldprt"))
	if err != nil {
		t.Fatalf("Open format B: %v", err)
	}
	if d.Version == 0 {
		t.Error("format-B version not parsed")
	}
	cfg, err := d.Stream("Contents/Config-0")
	if err != nil {
		t.Fatalf("Config-0: %v", err)
	}
	// MFC CArchive new-class tag (ff ff), schema 1, name-length 8, "moPart_c".
	if !bytes.Contains(cfg[:min(64, len(cfg))], []byte("moPart_c")) {
		t.Errorf("Config-0 does not start with the moPart_c CArchive header; head=% x", cfg[:min(32, len(cfg))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
