//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The M05-F09 feedback surfaces: balloon-tip toasts, the prompt modal, and the
// message-center panel. All three read session state each frame and write user
// gestures back through the session, so their owners observe the push events.

// drawMessagingSurfaces renders the active balloons, a pending prompt, and the
// Messages panel when open.
func drawMessagingSurfaces(s *app.Session) {
	drawBalloonToasts(s)
	drawPromptModal(s)
	drawMessageCenter(s)
}

// drawBalloonToasts renders each showing balloon as a small fixed window stacked
// up the bottom-right corner. Clicking the body reports the click; the X closes;
// the checkbox suppresses the tip for good.
func drawBalloonToasts(s *app.Session) {
	const toastW, toastH, margin = 320, 110, 16
	screenW, screenH := native.MainViewportSize()
	for i, tip := range s.BalloonTips().Active() {
		x := screenW - toastW - margin
		y := screenH - float32(i+1)*(toastH+margin)
		native.SetNextWindowPos(x, y)
		native.SetNextWindowSize(toastW, toastH)
		visible, open := native.BeginClosable(tip.Title + "###balloon-" + tip.ID)
		if visible {
			if native.Selectable(tip.Text, false) {
				s.ClickBalloonTip(tip.ID)
			}
			dontShow := false
			if native.Checkbox("Don't show again##"+tip.ID, &dontShow) && dontShow {
				s.DismissBalloonTip(tip.ID, true)
			}
		}
		native.End()
		if !open {
			s.DismissBalloonTip(tip.ID, false)
		}
	}
}

// drawPromptModal renders the oldest pending prompt as a centered modal: the
// message, one button per answer, and the remember checkbox when allowed.
func drawPromptModal(s *app.Session) {
	spec, ok := s.Prompts().Pending()
	if !ok {
		return
	}
	screenW, screenH := native.MainViewportSize()
	native.SetNextWindowPos(screenW/2-220, screenH/2-80)
	native.SetNextWindowSize(440, 0)
	if native.Begin("Prompt###prompt-modal") {
		native.Text(spec.Message)
		native.Separator()
		if spec.Restriction == types.PromptAllowRemember {
			native.Checkbox("Remember my answer", &promptRemember)
		}
		for i, label := range spec.Buttons {
			if i > 0 {
				native.SameLine()
			}
			if native.Button(label + "##prompt-" + strconv.Itoa(i)) {
				_ = s.AnswerPrompt(spec.ID, label, promptRemember)
				promptRemember = false
			}
		}
	}
	native.End()
}

// promptRemember is the modal's "remember my answer" checkbox state (UI state, so
// it lives in the head; reset after each answer).
var promptRemember bool

// drawMessageCenter renders the sectioned message tree (errors.show / the status
// badge open it).
func drawMessageCenter(s *app.Session) {
	if !s.MessageCenterOpen() {
		return
	}
	native.SetNextWindowSizeOnce(420, 320)
	visible, open := native.BeginClosable("Messages###message-center")
	if visible {
		if native.Button("Clear") {
			s.Messages().Clear()
		}
		native.Separator()
		drawMessageSection(s.Messages().View())
	}
	native.End()
	if !open {
		s.SetMessageCenterOpen(false)
	}
}

// drawMessageSection renders one section's messages and nested sections.
func drawMessageSection(section wire.MessageSectionView) {
	for _, msg := range section.Messages {
		native.BulletText(severityPrefix(msg.Severity) + msg.Text)
	}
	for _, sub := range section.Sections {
		native.SetNextItemOpen(true, true)
		if native.TreeNode(sub.Title) {
			drawMessageSection(sub)
			native.TreePop()
		}
	}
}

// severityPrefix tags a message row by its severity.
func severityPrefix(sev types.MessageSeverity) string {
	switch sev {
	case types.SeverityError:
		return "[error] "
	case types.SeverityWarning:
		return "[warning] "
	default:
		return ""
	}
}
