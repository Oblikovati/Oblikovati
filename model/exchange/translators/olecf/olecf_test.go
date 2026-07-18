// SPDX-License-Identifier: GPL-2.0-only

package olecf

import (
	"os"
	"path/filepath"
	"testing"
)

// sample.cfb is a real compound file (an Inventor .ipt is one) used purely to exercise
// container parsing; olecf itself is format-agnostic.
func openSample(t *testing.T) *File {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "sample.cfb"))
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}
	f, err := Open(data)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return f
}

func TestOpenParsesRootAndStreams(t *testing.T) {
	f := openSample(t)
	if f.RootCLSID() == [16]byte{} {
		t.Errorf("root CLSID is all-zero; expected a document class id")
	}
	if len(f.Streams()) == 0 {
		t.Fatalf("no streams enumerated")
	}
}

// TestReadHandlesBothFATAndMiniFAT reads a tiny stream (mini-FAT-backed) and a >4KB
// stream (FAT-backed) so both storage paths are exercised.
func TestReadHandlesBothFATAndMiniFAT(t *testing.T) {
	f := openSample(t)
	var small, big string
	for _, p := range f.Streams() {
		b, err := f.Read(p)
		if err != nil {
			t.Fatalf("Read %q: %v", p, err)
		}
		if len(b) > 0 && len(b) < 100 && small == "" {
			small = p
		}
		if len(b) > 4096 && big == "" {
			big = p
		}
	}
	if small == "" {
		t.Errorf("no small (mini-FAT) stream found to exercise the mini path")
	}
	if big == "" {
		t.Errorf("no >4KB (FAT) stream found to exercise the FAT path")
	}
}
