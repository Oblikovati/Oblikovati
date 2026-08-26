// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"oblikovati.org/model/compdef"
)

// Compile-time proof that the migrated tools drive the slim host, not *Session (#1635).
var (
	_ hostedTool = (*ExtrudeTool)(nil)
	_ hostedTool = (*FilletTool)(nil)
	_ hostedTool = (*ChamferTool)(nil)
)

// fakeToolHost is a named fake ToolHost (not the real *Session) that hands a tool a
// prebuilt part and records the label its commit passes to recordEdit — enough to run a
// tool's create-commit under test without constructing the whole app (#1635). It is the
// evidence for the return-position caveat on ToolHost.ActivePart: the "fake" still holds
// a real *compdef.PartComponentDefinition, because the return type is concrete.
type fakeToolHost struct {
	part         *compdef.PartComponentDefinition
	partErr      error
	recordedFrom recipeStore
	recordedAs   string
}

func (h *fakeToolHost) ActivePart() (*compdef.PartComponentDefinition, error) {
	return h.part, h.partErr
}

func (h *fakeToolHost) recordEdit(content recipeStore, label string) {
	h.recordedFrom, h.recordedAs = content, label
}

// hostFromBlock builds a real part with a block and wraps it in a fake host — the create
// path needs a real part (the ActivePart return-type caveat), so we reuse the block helper
// and read the part back off the session it seeds.
func hostFromBlock(t *testing.T, side float64) (*fakeToolHost, *compdef.PartComponentDefinition) {
	t.Helper()
	s, _ := newPartWithBlock(t, side)
	part := activePartDef(t, s)
	return &fakeToolHost{part: part}, part
}

// TestFilletCommitFeatureUsesHost drives the fillet create-commit against a fake host —
// pick an edge, set a radius, CanCommit, CommitFeature — and asserts the feature landed on
// the fake's part and the undo label was recorded through the host, never a *Session.
func TestFilletCommitFeatureUsesHost(t *testing.T) {
	host, part := hostFromBlock(t, 2)
	before := part.Features().Count()
	tool := NewFilletTool()
	tool.Pick(nil, verticalEdgeOf(t, part.SurfaceBodies().Item(0))) // Pick ignores the session
	tool.SetRadius(0.5)
	if !tool.CanCommit() {
		t.Fatal("fillet not ready after edge + radius")
	}
	if err := tool.CommitFeature(host); err != nil {
		t.Fatalf("CommitFeature: %v", err)
	}
	if got := part.Features().Count(); got != before+1 {
		t.Fatalf("feature count = %d, want %d (fillet added through host)", got, before+1)
	}
	if host.recordedAs != "Fillet" || host.recordedFrom != part {
		t.Fatalf("recordEdit(%v, %q), want (part, \"Fillet\")", host.recordedFrom, host.recordedAs)
	}
}

// TestChamferCommitFeatureUsesHost is the fillet test's sibling for chamfer — the third
// converted tool — proving the host seam generalizes across the dress-up family.
func TestChamferCommitFeatureUsesHost(t *testing.T) {
	host, part := hostFromBlock(t, 2)
	tool := NewChamferTool()
	tool.Pick(nil, verticalEdgeOf(t, part.SurfaceBodies().Item(0)))
	tool.SetDistance(0.3)
	if !tool.CanCommit() {
		t.Fatal("chamfer not ready after edge + distance")
	}
	if err := tool.CommitFeature(host); err != nil {
		t.Fatalf("CommitFeature: %v", err)
	}
	if host.recordedAs != "Chamfer" {
		t.Fatalf("recordEdit label = %q, want \"Chamfer\"", host.recordedAs)
	}
}

// TestCommitFeaturePropagatesHostError proves the host is the sole source of the active
// part: for every converted tool, a host that errors makes the commit error without touching
// a session (the return-err branch each CommitFeature shares, #1635).
func TestCommitFeaturePropagatesHostError(t *testing.T) {
	sentinel := errors.New("fake host: no active part")
	host := &fakeToolHost{partErr: sentinel}
	tools := []hostedTool{NewFilletTool(), NewChamferTool(), NewExtrudeTool()}
	for _, tool := range tools {
		if err := tool.CommitFeature(host); err != sentinel {
			t.Errorf("%T.CommitFeature error = %v, want the host's active-part error", tool, err)
		}
	}
	if host.recordedAs != "" {
		t.Errorf("a host-error commit recorded edit %q, want none", host.recordedAs)
	}
}

// TestFilletCommitFeatureSickReturnsError: driving CommitFeature directly (the host seam bypasses
// the preview sick-config gate) with a radius that overruns the block builds a sick feature, so
// the commit returns the health reason to keep the tool open (#1635).
func TestFilletCommitFeatureSickReturnsError(t *testing.T) {
	host, part := hostFromBlock(t, 2)
	tool := NewFilletTool()
	tool.Pick(nil, verticalEdgeOf(t, part.SurfaceBodies().Item(0)))
	tool.SetRadius(10) // the rolling ball overruns the 2×2×2 block ⇒ sick
	err := tool.CommitFeature(host)
	if err == nil || !strings.HasPrefix(err.Error(), "fillet: ") {
		t.Fatalf("sick fillet CommitFeature err = %v, want a \"fillet: \" health reason", err)
	}
}

// hostConvertedTools is the ratchet's tracking list: every part-feature tool whose
// create-commit has migrated to ToolHost. It only grows — a tool is added here as it is
// converted, never removed. Typed []Tool (not []hostedTool) so the ratchet's hostedTool
// check is a real assertion, not a tautology. The remaining ~40 PartFeatureTools still
// route their create path through *Session and are the backlog this list is drawn from
// (#1635, no big bang).
var hostConvertedTools = []Tool{
	(*ExtrudeTool)(nil),
	(*FilletTool)(nil),
	(*ChamferTool)(nil),
}

// hostConvertedCount pins how many tools have been converted. Raise it only when adding a
// tool to hostConvertedTools; it must never decrease (that would mean a tool regressed off
// the host seam). This is the I12 ratchet.
const hostConvertedCount = 3

func TestHostedToolRatchet(t *testing.T) {
	if len(hostConvertedTools) != hostConvertedCount {
		t.Fatalf("converted-tool ratchet: %d tools, pinned %d — update hostConvertedCount only upward (#1635)",
			len(hostConvertedTools), hostConvertedCount)
	}
	for _, tool := range hostConvertedTools {
		if _, ok := tool.(hostedTool); !ok {
			t.Errorf("%T is listed converted but does not implement hostedTool", tool)
		}
	}
}

// TestToolHostStaysSlim pins ToolHost's method ceiling so it cannot silently re-fatten
// into a second Session (#1635). Grow the ceiling only with deliberate evidence.
func TestToolHostStaysSlim(t *testing.T) {
	const ceiling = 8
	got := reflect.TypeFor[ToolHost]().NumMethod()
	if got > ceiling {
		t.Fatalf("ToolHost has %d methods, ceiling %d — a slim host must not re-fatten (#1635)", got, ceiling)
	}
}
