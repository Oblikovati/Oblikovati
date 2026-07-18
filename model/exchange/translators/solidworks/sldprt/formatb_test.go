// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import (
	"bytes"
	"testing"
)

// TestNibbleSwapDecodesStreamName verifies the format-B name obfuscation: the raw stored bytes of
// "Contents" (each byte nibble-swapped) round-trip back to the ASCII name, and the swap is its own
// inverse.
func TestNibbleSwapDecodesStreamName(t *testing.T) {
	stored := []byte{0x34, 0xf6, 0xe6, 0x47, 0x56, 0xe6, 0x47, 0x37}
	if got := string(nibbleSwap(stored)); got != "Contents" {
		t.Fatalf("nibbleSwap = %q, want Contents", got)
	}
	if !bytes.Equal(nibbleSwap(nibbleSwap(stored)), stored) {
		t.Error("nibbleSwap is not an involution")
	}
}

// TestParseFormatBExtractsConfig0 verifies the record log parses out every stream and that
// Config-0 inflates to the MFC CArchive graph shared with format A.
func TestParseFormatBExtractsConfig0(t *testing.T) {
	streams, err := parseFormatB(readTestdata(t, "perno_fmtb.sldprt"))
	if err != nil {
		t.Fatalf("parseFormatB: %v", err)
	}
	cfg, ok := streams["Contents/Config-0"]
	if !ok {
		t.Fatalf("Config-0 not extracted; got %d streams", len(streams))
	}
	if !bytes.Contains(cfg[:min(64, len(cfg))], []byte("moPart_c")) {
		t.Errorf("Config-0 is not the CArchive graph; head=% x", cfg[:min(32, len(cfg))])
	}
	// The Parasolid partition must also come through (geometry path).
	if _, ok := streams["Contents/Config-0-Partition"]; !ok {
		t.Error("Config-0-Partition stream missing")
	}
}
