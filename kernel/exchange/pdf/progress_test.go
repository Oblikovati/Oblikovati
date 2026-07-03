// SPDX-License-Identifier: GPL-2.0-only

package pdf

import (
	"errors"
	"testing"

	"oblikovati.org/kernel/exchange"
)

// pdfProgressSink records the progress ticks the PDF decoder reports (a named fake per the house
// rules).
type pdfProgressSink struct{ pages int }

func (p *pdfProgressSink) fn(string, int, int) bool { p.pages++; return false }

// TestDecodePagesWithProgressReports checks the PDF decoder threads the shared progress seam
// (#1647): it fires a per-page tick. Skips when the PDF corpus is unavailable.
func TestDecodePagesWithProgressReports(t *testing.T) {
	var sink pdfProgressSink
	if _, _, err := DecodePagesWithProgress(corpusFile(t, "pdf-vector2.pdf"), exchange.TranslationOptions{Progress: sink.fn}); err != nil {
		t.Fatalf("DecodePagesWithProgress: %v", err)
	}
	if sink.pages == 0 {
		t.Fatal("the PDF decoder reported no progress; the seam is not wired")
	}
}

// TestDecodePagesWithProgressCancels checks a first-tick cancel aborts the PDF import with an
// ErrCancelled-wrapping error. Skips when the PDF corpus is unavailable.
func TestDecodePagesWithProgressCancels(t *testing.T) {
	cancel := func(string, int, int) bool { return true }
	_, _, err := DecodePagesWithProgress(corpusFile(t, "pdf-vector2.pdf"), exchange.TranslationOptions{Progress: cancel})
	if !errors.Is(err, exchange.ErrCancelled) {
		t.Fatalf("cancelled PDF import error = %v, want it to wrap exchange.ErrCancelled", err)
	}
}
