//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/api/contract"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/doc"
	"oblikovati.org/renderer"
)

// These tests exercise the audit-I5 consumer interfaces: each dialog now names the slim
// session surface it consumes, so it can be driven by a NAMED fake host (CLAUDE.md: named
// fake classes, not inline stubs) instead of the whole *app.Session — the coupling that
// previously made no widget testable in isolation.

// fakeCommitHost is a fake commitCancelHost recording the OK/Cancel it receives, so a test
// can drive the shared button row without a real session or model.
type fakeCommitHost struct {
	blocked string
	okCalls int
	cancels int
}

func (h *fakeCommitHost) CommitBlockedReason() string { return h.blocked }
func (h *fakeCommitHost) OK() error                   { h.okCalls++; return nil }
func (h *fakeCommitHost) CancelTool()                 { h.cancels++ }

var _ commitCancelHost = (*fakeCommitHost)(nil)

// fakeMeasureHost is a fake measureHost carrying a real MeasureTool but stubbing the
// commit controls — the panel needs no session.
type fakeMeasureHost struct {
	fakeCommitHost
	measure *app.MeasureTool
}

func (h *fakeMeasureHost) ActiveMeasure() *app.MeasureTool { return h.measure }

var _ measureHost = (*fakeMeasureHost)(nil)

// fakeGripSnapHost is a fake gripSnapHost carrying a real GripSnapTool.
type fakeGripSnapHost struct {
	fakeCommitHost
	grip *app.GripSnapTool
}

func (h *fakeGripSnapHost) ActiveGripSnap() *app.GripSnapTool { return h.grip }

var _ gripSnapHost = (*fakeGripSnapHost)(nil)

// fakeEdgeColorSource is a fake edgeColorSource with no active document — displayEdgeColor
// must fall back to the renderer default without a session.
type fakeEdgeColorSource struct{}

func (fakeEdgeColorSource) ActiveDocument() *doc.Document { return nil }
func (fakeEdgeColorSource) DocumentDisplaySettings(doc.ID) contract.DisplaySettings {
	return nil
}

var (
	_ edgeColorSource      = fakeEdgeColorSource{}
	_ activeDocumentSource = fakeEdgeColorSource{}
)

// TestDisplayEdgeColorFallsBackWithoutDocument drives displayEdgeColor from a fake host with
// no active document (a pure, native-free unit test enabled by the consumer interface).
func TestDisplayEdgeColorFallsBackWithoutDocument(t *testing.T) {
	if got := displayEdgeColor(fakeEdgeColorSource{}); got != renderer.DefaultEdgeColor() {
		t.Fatalf("displayEdgeColor with no document = %v, want renderer default %v", got, renderer.DefaultEdgeColor())
	}
}

// TestMeasurePanelDrivenByFakeHost renders the Measure panel from a fake host — proving the
// panel consumes only measureHost, not *app.Session.
func TestMeasurePanelDrivenByFakeHost(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	icons = nil

	win.BeginFrame()
	drawMeasure(&fakeMeasureHost{}) // no active tool → early return, draws nothing
	drawMeasure(&fakeMeasureHost{measure: app.NewMeasureTool()})
	win.EndFrame(0.1, 0.1, 0.1)
}

// TestGripSnapPanelDrivenByFakeHost renders the Grip Snap panel from a fake host.
func TestGripSnapPanelDrivenByFakeHost(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	icons = nil

	win.BeginFrame()
	drawGripSnap(&fakeGripSnapHost{}) // no active tool → early return
	drawGripSnap(&fakeGripSnapHost{grip: app.NewGripSnapTool()})
	win.EndFrame(0.1, 0.1, 0.1)
}

// TestCommitCancelButtonsDrivenByFakeHost renders the shared OK/Cancel row from a fake host,
// the widget every registered tool dialog draws through.
func TestCommitCancelButtonsDrivenByFakeHost(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	icons = nil

	win.BeginFrame()
	if native.Begin("commit-probe") {
		drawCommitCancelButtons(&fakeCommitHost{}, false)
		drawCommitCancelButtons(&fakeCommitHost{blocked: "would recompute sick"}, true)
	}
	native.End()
	win.EndFrame(0.1, 0.1, 0.1)
}
