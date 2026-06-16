// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"bytes"
	"testing"
)

// The AcDb:Header section opens with vbeginSentinel (defined in headervars.go) in every
// DWG generation; finding it at the front of the assembled section proves the whole paged
// pipeline (decrypt → page map → section map → data-page decompress → assemble) is correct.

// r2018Corpus is every AC1032 file in the test corpus; the AC1015 file is covered
// by the R2000 tests.
var r2018Corpus = []string{
	"testfile-1.dwg", "testfile-3.dwg", "testfile-4.dwg",
	"testfile-5.dwg", "testfile-6.dwg", "testfile-7.dwg",
}

// essentialSections are the logical sections the entity decoder needs; every well
// formed drawing must carry them.
var essentialSections = []string{
	"AcDb:Header", "AcDb:Classes", "AcDb:Handles", "AcDb:AcDbObjects",
}

// TestParseR2004ContainerCorpus runs the whole paged-container pipeline on every
// AC1032 file: detect version, decrypt the header, build the page map, parse the
// section map, then assemble each essential logical section and confirm it is
// non-empty. This is the end-to-end gate for the R2004+ container.
func TestParseR2004ContainerCorpus(t *testing.T) {
	for _, name := range r2018Corpus {
		t.Run(name, func(t *testing.T) {
			data := loadTestFile(t, name)
			h, err := ParseFileHeader(data)
			if err != nil {
				t.Fatalf("ParseFileHeader: %v", err)
			}
			if h.Version != R2018 {
				t.Fatalf("version = %v, want R2018", h.Version)
			}
			present := map[string]bool{}
			for _, n := range h.SectionNames() {
				present[n] = true
			}
			for _, must := range essentialSections {
				if !present[must] {
					t.Errorf("missing section %q (have %v)", must, h.SectionNames())
				}
			}
			for _, must := range essentialSections {
				sec, err := h.LogicalSection(data, must)
				if err != nil {
					t.Errorf("assemble %q: %v", must, err)
					continue
				}
				if len(sec) == 0 {
					t.Errorf("section %q assembled to 0 bytes", must)
				}
			}
			// The header section must open with the begin sentinel — a byte-exact
			// check that the data-page decompression produced the real content.
			hdr, err := h.LogicalSection(data, "AcDb:Header")
			if err != nil {
				t.Fatalf("assemble AcDb:Header: %v", err)
			}
			if !bytes.HasPrefix(hdr, vbeginSentinel) {
				t.Errorf("AcDb:Header does not start with begin sentinel; got % X", hdr[:min(16, len(hdr))])
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestLogicalSectionOnR2000Rejected guards the version branch: asking a flat R2000
// file for a named logical section is a programming error and must be reported.
func TestLogicalSectionOnR2000Rejected(t *testing.T) {
	data := loadTestFile(t, "testfile-2.dwg")
	h, err := ParseFileHeader(data)
	if err != nil {
		t.Fatalf("ParseFileHeader: %v", err)
	}
	if _, err := h.LogicalSection(data, "AcDb:Header"); err == nil {
		t.Fatal("expected error requesting a logical section from an R2000 file")
	}
}
