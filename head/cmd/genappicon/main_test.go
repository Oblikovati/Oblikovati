// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGeneratePNG(t *testing.T) {
	out := filepath.Join(t.TempDir(), "icon.png")
	if err := generate("png", 48, "", out); err != nil {
		t.Fatalf("generate png: %v", err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.Width != 48 {
		t.Fatalf("width = %d, want 48", cfg.Width)
	}
}

func TestGenerateICO(t *testing.T) {
	out := filepath.Join(t.TempDir(), "icon.ico")
	if err := generate("ico", 0, "16,32", out); err != nil {
		t.Fatalf("generate ico: %v", err)
	}
	if fi, err := os.Stat(out); err != nil || fi.Size() == 0 {
		t.Fatalf("ico not written: stat=%v err=%v", fi, err)
	}
}

func TestGenerateRejectsBadFormatAndMissingOut(t *testing.T) {
	if err := generate("png", 32, "", ""); err == nil {
		t.Fatal("missing -out should error")
	}
	if err := generate("bmp", 32, "", filepath.Join(t.TempDir(), "x")); err == nil {
		t.Fatal("unknown format should error")
	}
}

func TestParseSizes(t *testing.T) {
	got, err := parseSizes("16, 32 ,256")
	if err != nil {
		t.Fatalf("parseSizes: %v", err)
	}
	if want := []int{16, 32, 256}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if _, err := parseSizes("16,x"); err == nil {
		t.Fatal("non-numeric size should error")
	}
}
