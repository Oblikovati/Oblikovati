// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/app/cmdline"
)

// scrollbackHas reports whether the command line shows a line with the given text and
// severity — the assertion for "this message reached the Command Window".
func scrollbackHas(s *Session, text string, sev cmdline.Severity) bool {
	for _, l := range s.CommandLine().Scrollback().Lines() {
		if l.Text == text && l.Severity == sev {
			return true
		}
	}
	return false
}

func TestNoticeFunnelsToCommandLine(t *testing.T) {
	s := NewSession()
	s.SetNotice("commit failed: open profile")
	if !scrollbackHas(s, "commit failed: open profile", cmdline.Info) {
		t.Error("SetNotice did not appear in the Command Window")
	}
}

func TestMessageCenterFunnelsToCommandLine(t *testing.T) {
	s := NewSession()
	s.Messages().AddMessage("boolean failed", types.SeverityError)
	s.Messages().AddMessage("degenerate face", types.SeverityWarning)
	if !scrollbackHas(s, "boolean failed", cmdline.Error) {
		t.Error("error message did not funnel with Error severity")
	}
	if !scrollbackHas(s, "degenerate face", cmdline.Warning) {
		t.Error("warning message did not funnel with Warning severity")
	}
}

func TestBalloonTipFunnelsToCommandLine(t *testing.T) {
	s := NewSession()
	if err := s.BalloonTips().Register(BalloonTipSpec{ID: "tip1", Title: "Saved", Text: "to disk"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if _, err := s.ShowBalloonTip("tip1"); err != nil {
		t.Fatalf("ShowBalloonTip: %v", err)
	}
	if !scrollbackHas(s, "Saved — to disk", cmdline.Info) {
		t.Error("balloon tip did not funnel to the Command Window")
	}
}

func TestPromptAskedAndAnsweredViaCommandLine(t *testing.T) {
	s := NewSession()
	spec := PromptSpec{ID: "overwrite", Message: "Overwrite the file?", Buttons: []string{"Yes", "No"}}
	if _, _, err := s.ShowPrompt(spec); err != nil {
		t.Fatalf("ShowPrompt: %v", err)
	}
	if !scrollbackHas(s, "Overwrite the file? [Yes/No]", cmdline.Prompt) {
		t.Error("prompt question was not asked on the command line")
	}
	if !s.CommandLine().Awaiting(s) {
		t.Error("engine should be awaiting the prompt answer")
	}
	if err := s.CommandLine().Submit(s, "y"); err != nil { // prefix match → "Yes"
		t.Fatalf("answer submit: %v", err)
	}
	if _, pending := s.Prompts().Pending(); pending {
		t.Error("prompt should be resolved after a valid answer")
	}
	if !scrollbackHas(s, "Yes", cmdline.Echo) {
		t.Error("the answer was not echoed")
	}
}

func TestPromptEmptyAcceptsDefault(t *testing.T) {
	s := NewSession()
	spec := PromptSpec{ID: "q", Message: "Proceed?", Buttons: []string{"Cancel", "OK"}, Default: 1}
	if _, _, err := s.ShowPrompt(spec); err != nil {
		t.Fatalf("ShowPrompt: %v", err)
	}
	if err := s.CommandLine().Submit(s, ""); err != nil { // Enter → default button "OK"
		t.Fatalf("default answer: %v", err)
	}
	if _, pending := s.Prompts().Pending(); pending {
		t.Error("empty submit should accept the default and resolve the prompt")
	}
	if !scrollbackHas(s, "OK", cmdline.Echo) {
		t.Error("default answer OK was not echoed")
	}
}

func TestPromptUnrecognizedAnswerReasks(t *testing.T) {
	s := NewSession()
	spec := PromptSpec{ID: "q", Message: "Proceed?", Buttons: []string{"Yes", "No"}}
	if _, _, err := s.ShowPrompt(spec); err != nil {
		t.Fatalf("ShowPrompt: %v", err)
	}
	_ = s.CommandLine().Submit(s, "maybe") // not a button
	if _, pending := s.Prompts().Pending(); !pending {
		t.Error("an unrecognised answer must leave the prompt pending")
	}
}
