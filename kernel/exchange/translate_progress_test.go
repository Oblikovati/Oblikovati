// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"errors"
	"testing"
)

// TestReportNilProgressIsNoOp pins the nil-safety contract (#1647): the zero TranslationOptions has
// no Progress, so Report never calls back and never cancels — every importer runs unchanged when no
// sink is wired.
func TestReportNilProgressIsNoOp(t *testing.T) {
	var opts TranslationOptions
	if err := opts.Report("stage", 3, 10); err != nil {
		t.Errorf("Report with a nil Progress must be a no-op, got %v", err)
	}
}

// TestReportForwardsTick checks Report passes the stage and counts straight through to the sink and
// returns nil when the sink does not cancel.
func TestReportForwardsTick(t *testing.T) {
	var got struct {
		stage       string
		done, total int
	}
	opts := TranslationOptions{Progress: func(stage string, done, total int) bool {
		got.stage, got.done, got.total = stage, done, total
		return false
	}}
	if err := opts.Report("entities", 4, 9); err != nil {
		t.Fatalf("Report returned %v, want nil (no cancel)", err)
	}
	if got.stage != "entities" || got.done != 4 || got.total != 9 {
		t.Errorf("sink saw %q %d/%d, want \"entities\" 4/9", got.stage, got.done, got.total)
	}
}

// TestReportCancelWrapsErrCancelled checks a sink asking to cancel yields a stage-naming error that
// unwraps to ErrCancelled, so callers can tell a user cancel from a decode failure with errors.Is.
func TestReportCancelWrapsErrCancelled(t *testing.T) {
	opts := TranslationOptions{Progress: func(string, int, int) bool { return true }}
	err := opts.Report("points", 1, 100)
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("cancel Report error = %v, want it to wrap ErrCancelled", err)
	}
}
