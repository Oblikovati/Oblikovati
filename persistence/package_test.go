// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"
)

func packageWith(streams map[string][]byte) *Package {
	p := NewPackage()
	for name, data := range streams {
		p.WriteStream(name, data)
	}
	return p
}

func TestStreamsRoundTripByteIdentical(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "model.obk")
	want := map[string][]byte{
		"model/parameters.bin": {0x00, 0x01, 0xfe, 0xff, 0x10},
		"thumbnail.png":        []byte("not really a png"),
	}
	if err := packageWith(want).Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := OpenPackage(path)
	if err != nil {
		t.Fatalf("OpenPackage: %v", err)
	}
	for name, data := range want {
		back, ok := got.ReadStream(name)
		if !ok {
			t.Errorf("stream %q missing after round trip", name)
			continue
		}
		if !bytes.Equal(back, data) {
			t.Errorf("stream %q = %v, want byte-identical %v", name, back, data)
		}
	}
}

func TestWriteStreamCopiesBytes(t *testing.T) {
	src := []byte{1, 2, 3}
	p := NewPackage()
	p.WriteStream("s", src)
	src[0] = 99 // mutate caller's slice after writing
	back, _ := p.ReadStream("s")
	if back[0] != 1 {
		t.Errorf("package shares backing array with caller: got %v", back)
	}
}

func TestStreamsAndStat(t *testing.T) {
	p := packageWith(map[string][]byte{"a.bin": {1, 2}, "b.bin": {1, 2, 3}})
	if got := len(p.Streams()); got != 2 {
		t.Fatalf("Streams len = %d, want 2", got)
	}
	st, ok := p.Stat("b.bin")
	if !ok || st.Size != 3 {
		t.Errorf("Stat(b.bin) = %+v ok=%v, want size 3", st, ok)
	}
	if _, ok := p.Stat("missing"); ok {
		t.Error("Stat reported a missing stream as present")
	}
}

func TestDeleteStream(t *testing.T) {
	p := packageWith(map[string][]byte{"a": {1}})
	if !p.DeleteStream("a") {
		t.Error("DeleteStream returned false for an existing stream")
	}
	if p.DeleteStream("a") {
		t.Error("DeleteStream returned true for an already-removed stream")
	}
	if _, ok := p.ReadStream("a"); ok {
		t.Error("stream still readable after delete")
	}
}

func TestManifestRoundTripAndAbsence(t *testing.T) {
	p := NewPackage()
	if _, err := p.Manifest(); !errors.Is(err, errNoManifest) {
		t.Errorf("Manifest on empty package = %v, want errNoManifest", err)
	}
	want := Manifest{SchemaVersion: CurrentSchemaVersion, DocumentType: 1, DisplayName: "Part1"}
	if err := p.SetManifest(want); err != nil {
		t.Fatalf("SetManifest: %v", err)
	}
	got, err := p.Manifest()
	if err != nil || got != want {
		t.Errorf("Manifest round trip = %+v err=%v, want %+v", got, err, want)
	}
}

func TestInterruptedSaveLeavesPriorFileIntact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.obk")

	v1, err := packageWith(map[string][]byte{"s": []byte("v1")}).marshal()
	if err != nil {
		t.Fatalf("marshal v1: %v", err)
	}
	if err := atomicWriteFile(path, v1); err != nil {
		t.Fatalf("write v1: %v", err)
	}

	// Stage v2 but never commit — the "crash" happens between stage and rename.
	v2, _ := packageWith(map[string][]byte{"s": []byte("v2")}).marshal()
	tmp, err := stage(path, v2)
	if err != nil {
		t.Fatalf("stage v2: %v", err)
	}

	// The live file is still the intact v1; the staged temp is ignored.
	prior, err := OpenPackage(path)
	if err != nil {
		t.Fatalf("OpenPackage after staged-but-uncommitted save: %v", err)
	}
	if got, _ := prior.ReadStream("s"); string(got) != "v1" {
		t.Errorf("prior file corrupted by interrupted save: %q, want v1", got)
	}

	// Completing the rename swaps in v2 atomically.
	if err := commit(tmp, path); err != nil {
		t.Fatalf("commit: %v", err)
	}
	next, _ := OpenPackage(path)
	if got, _ := next.ReadStream("s"); string(got) != "v2" {
		t.Errorf("after commit: %q, want v2", got)
	}
}
