// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"maps"
	"slices"

	"oblikovati.org/app/cmdline"
	"oblikovati.org/event"
	"oblikovati.org/persistence/dialogmemory"
)

// BalloonTipClicked fires (After) when the user clicks a balloon's body; the events
// relay forwards it as a balloonTip.clicked push event (M05-F09).
type BalloonTipClicked struct{ ID string }

// EventID implements event.Event.
func (BalloonTipClicked) EventID() event.TypeID { return tidBalloonClicked }

// BalloonTipSpec is one registered notification balloon.
type BalloonTipSpec struct {
	ID    string
	Title string
	Text  string
	Icon  string
}

// BalloonTipCenter registers named balloons and tracks the live (showing) ones —
// the BalloonTip(s) equivalent. Suppression ("don't show again") persists per user
// through the dialog-memory store.
type BalloonTipCenter struct {
	specs      map[string]BalloonTipSpec
	active     []string
	suppressed map[string]bool
}

// NewBalloonTipCenter returns an empty balloon registry.
func NewBalloonTipCenter() *BalloonTipCenter {
	return &BalloonTipCenter{specs: map[string]BalloonTipSpec{}, suppressed: map[string]bool{}}
}

// Register declares (or redeclares) a balloon under its stable id.
func (b *BalloonTipCenter) Register(spec BalloonTipSpec) error {
	if spec.ID == "" || spec.Title == "" {
		return fmt.Errorf("app: balloon tip needs id and title, got id=%q title=%q", spec.ID, spec.Title)
	}
	b.specs[spec.ID] = spec
	return nil
}

// Active returns the currently showing balloons in show order.
func (b *BalloonTipCenter) Active() []BalloonTipSpec {
	out := make([]BalloonTipSpec, len(b.active))
	for i, id := range b.active {
		out[i] = b.specs[id]
	}
	return out
}

// BalloonTips returns the session's balloon registry.
func (s *Session) BalloonTips() *BalloonTipCenter { return s.balloonTips }

// ShowBalloonTip displays a registered balloon; it reports false (without showing)
// when the user suppressed it, mirroring the wire reply.
func (s *Session) ShowBalloonTip(id string) (bool, error) {
	b := s.balloonTips
	if _, ok := b.specs[id]; !ok {
		return false, fmt.Errorf("app: no balloon tip %q (register it first)", id)
	}
	if b.suppressed[id] {
		return false, nil
	}
	if slices.Contains(b.active, id) {
		return true, nil // already showing
	}
	b.active = append(b.active, id)
	s.feedScrollback(balloonLine(b.specs[id]), cmdline.Info) // M26 F03: also show in the Command Window
	return true, nil
}

// ClickBalloonTip reports the user clicked a showing balloon's body: it dismisses
// the balloon and emits the event its owner observes.
func (s *Session) ClickBalloonTip(id string) {
	s.dismissBalloon(id)
	event.Emit(s.bus, event.After, BalloonTipClicked{ID: id})
}

// DismissBalloonTip closes a showing balloon; dontShowAgain suppresses it for good
// (persisted per user).
func (s *Session) DismissBalloonTip(id string, dontShowAgain bool) {
	s.dismissBalloon(id)
	if dontShowAgain {
		s.balloonTips.suppressed[id] = true
		s.saveDialogMemory()
	}
}

// dismissBalloon removes a balloon from the active list.
func (s *Session) dismissBalloon(id string) {
	b := s.balloonTips
	for i, x := range b.active {
		if x == id {
			b.active = append(b.active[:i], b.active[i+1:]...)
			return
		}
	}
}

// UseDialogMemoryStore installs the per-user remembered-choice store and loads it
// into the balloon suppression set and the prompt answers.
func (s *Session) UseDialogMemoryStore(store dialogmemory.Store) error {
	mem, err := store.Load()
	if err != nil {
		return err
	}
	s.dialogMemoryStore = store
	for _, id := range mem.SuppressedTips {
		s.balloonTips.suppressed[id] = true
	}
	maps.Copy(s.prompts.remembered, mem.PromptAnswers)
	return nil
}

// saveDialogMemory persists the suppression set and remembered prompt answers.
func (s *Session) saveDialogMemory() {
	if s.dialogMemoryStore == nil {
		return
	}
	mem := dialogmemory.Memory{PromptAnswers: map[string]string{}}
	for id := range s.balloonTips.suppressed {
		mem.SuppressedTips = append(mem.SuppressedTips, id)
	}
	maps.Copy(mem.PromptAnswers, s.prompts.remembered)
	_ = s.dialogMemoryStore.Save(mem)
}
