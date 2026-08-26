// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"slices"

	"oblikovati.org/api/types"
	"oblikovati.org/event"
)

// PromptAnswered fires (After) when the user answers a pending prompt; the events
// relay forwards it as a prompt.answered push event (M05-F09).
type PromptAnswered struct {
	ID         string
	Answer     string
	Remembered bool
}

// EventID implements event.Event.
func (PromptAnswered) EventID() event.TypeID { return tidPromptAnswered }

// PromptSpec is one declarative prompt: the message, the answer buttons in display
// order, the default button index, and whether the user may remember their answer.
type PromptSpec struct {
	ID          string
	Message     string
	Buttons     []string
	Default     int
	Restriction types.PromptRestriction
}

// PromptCenter queues prompts for the head's modal and keeps the per-user
// remembered answers — the PromptMessage(s)/PromptsOptions equivalent. A wire call
// must never block the session goroutine on user input, so showing is asynchronous:
// remembered prompts resolve instantly, the rest resolve through [Session.AnswerPrompt].
type PromptCenter struct {
	pending    []PromptSpec
	remembered map[string]string
}

// NewPromptCenter returns an empty prompt queue.
func NewPromptCenter() *PromptCenter {
	return &PromptCenter{remembered: map[string]string{}}
}

// Pending returns the prompt the head should display (the oldest queued one).
func (p *PromptCenter) Pending() (PromptSpec, bool) {
	if len(p.pending) == 0 {
		return PromptSpec{}, false
	}
	return p.pending[0], true
}

// Prompts returns the session's prompt center.
func (s *Session) Prompts() *PromptCenter { return s.prompts }

// CancelPending discards every queued prompt without answering it — ESC cancels the whole
// operation that asked the question (M26), so its pending prompts are dropped, not defaulted.
func (p *PromptCenter) CancelPending() { p.pending = nil }

// ShowPrompt queues a prompt. A remembered answer resolves instantly (resolved true
// + the answer); otherwise the head shows the modal and the answer arrives through
// [Session.AnswerPrompt] as a [PromptAnswered] event.
func (s *Session) ShowPrompt(spec PromptSpec) (resolved bool, answer string, err error) {
	if spec.ID == "" || spec.Message == "" || len(spec.Buttons) == 0 {
		return false, "", fmt.Errorf("app: prompt needs id, message and buttons, got id=%q message=%q buttons=%v",
			spec.ID, spec.Message, spec.Buttons)
	}
	if spec.Default < 0 || spec.Default >= len(spec.Buttons) {
		return false, "", fmt.Errorf("app: prompt default %d out of range of %d buttons", spec.Default, len(spec.Buttons))
	}
	if remembered, ok := s.prompts.remembered[spec.ID]; ok && spec.Restriction == types.PromptAllowRemember {
		return true, remembered, nil
	}
	for _, queued := range s.prompts.pending {
		if queued.ID == spec.ID {
			return false, "", nil // already pending; one modal will answer both callers
		}
	}
	s.prompts.pending = append(s.prompts.pending, spec)
	return false, "", nil
}

// AnswerPrompt resolves the oldest pending prompt with one of its button labels —
// the head's modal calls it. remember keeps the answer (when the prompt allows it)
// so the next ShowPrompt resolves instantly.
func (s *Session) AnswerPrompt(id, answer string, remember bool) error {
	spec, ok := s.peekPendingPrompt(id)
	if !ok {
		return fmt.Errorf("app: no pending prompt %q", id)
	}
	// Validate BEFORE removing: a rejected answer must leave the prompt pending
	// (removing first would silently swallow it — caught by the regression test).
	if !promptHasButton(spec, answer) {
		return fmt.Errorf("app: prompt %q has no button %q (one of %v)", id, answer, spec.Buttons)
	}
	s.removePendingPrompt(id)
	remembered := remember && spec.Restriction == types.PromptAllowRemember
	if remembered {
		s.prompts.remembered[id] = answer
		s.saveDialogMemory()
	}
	event.Emit(s.bus, event.After, PromptAnswered{ID: id, Answer: answer, Remembered: remembered})
	return nil
}

// peekPendingPrompt returns the pending prompt with the given id without removing it.
func (s *Session) peekPendingPrompt(id string) (PromptSpec, bool) {
	for _, spec := range s.prompts.pending {
		if spec.ID == id {
			return spec, true
		}
	}
	return PromptSpec{}, false
}

// removePendingPrompt drops the pending prompt with the given id.
func (s *Session) removePendingPrompt(id string) {
	for i, spec := range s.prompts.pending {
		if spec.ID == id {
			s.prompts.pending = append(s.prompts.pending[:i], s.prompts.pending[i+1:]...)
			return
		}
	}
}

// promptHasButton reports whether answer is one of the prompt's buttons.
func promptHasButton(spec PromptSpec, answer string) bool {
	return slices.Contains(spec.Buttons, answer)
}
