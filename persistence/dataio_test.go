// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestDataIOPersistsArbitraryData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "addin-data.obk")
	payload := []byte{0xde, 0xad, 0xbe, 0xef, 0x00, 0x7f}

	if err := WriteDataToFile(path, "addins/acme/state.bin", payload); err != nil {
		t.Fatalf("WriteDataToFile: %v", err)
	}
	got, err := ReadDataFromFile(path, "addins/acme/state.bin")
	if err != nil {
		t.Fatalf("ReadDataFromFile: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("round trip = %v, want %v", got, payload)
	}
}

func TestWriteDataToFilePreservesOtherStreams(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shared.obk")
	if err := WriteDataToFile(path, "a.bin", []byte("aaa")); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if err := WriteDataToFile(path, "b.bin", []byte("bbb")); err != nil {
		t.Fatalf("write b: %v", err)
	}
	a, err := ReadDataFromFile(path, "a.bin")
	if err != nil || string(a) != "aaa" {
		t.Errorf("first stream lost when writing second: %q err=%v", a, err)
	}
}

func TestReadMissingStreamErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.obk")
	_ = WriteDataToFile(path, "present.bin", []byte("x"))
	if _, err := ReadDataFromFile(path, "absent.bin"); err == nil {
		t.Error("ReadDataFromFile returned no error for an absent stream")
	}
}

func TestCompactionDropsCacheLosslessly(t *testing.T) {
	p := packageWith(map[string][]byte{
		"model/parameters.bin": []byte("recipe"),
		"cache/tessellation":   bytes.Repeat([]byte{0xaa}, 4096),
		"cache/preview":        bytes.Repeat([]byte{0xbb}, 1024),
	})
	before, _ := p.marshal()

	reclaimed := Compact(p)
	if reclaimed != 4096+1024 {
		t.Errorf("reclaimed = %d, want %d", reclaimed, 4096+1024)
	}
	if _, ok := p.ReadStream("cache/tessellation"); ok {
		t.Error("cache stream survived compaction")
	}
	if got, ok := p.ReadStream("model/parameters.bin"); !ok || string(got) != "recipe" {
		t.Error("recipe stream lost during compaction")
	}

	after, _ := p.marshal()
	if len(after) >= len(before) {
		t.Errorf("compacted archive (%d B) not smaller than original (%d B)", len(after), len(before))
	}
}
