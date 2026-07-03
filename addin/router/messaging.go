// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// registerMessagingHandlers wires the status / progress / balloon / prompt /
// message-center methods (M05-F09, #616).
func (r *Router) registerMessagingHandlers() {
	r.readOnly(wire.MethodStatusSetText, typed(setStatusText))
	r.readOnly(wire.MethodStatusGetText, getStatusText)
	r.readOnly(wire.MethodProgressBegin, typed(beginProgress))
	r.readOnly(wire.MethodProgressUpdate, typed(updateProgress))
	r.readOnly(wire.MethodProgressEnd, typed(endProgress))
	r.readOnly(wire.MethodBalloonTipRegister, typed(registerBalloonTip))
	r.readOnly(wire.MethodBalloonTipShow, typed(showBalloonTip))
	r.readOnly(wire.MethodPromptsShow, typed(showPrompt))
	r.readOnly(wire.MethodErrorsAddMessage, typed(addErrorMessage))
	r.readOnly(wire.MethodErrorsBeginSection, typed(beginMessageSection))
	r.readOnly(wire.MethodErrorsEndSection, typed(endMessageSection))
	r.readOnly(wire.MethodErrorsList, listErrors)
	r.readOnly(wire.MethodErrorsClear, clearErrors)
	r.readOnly(wire.MethodErrorsShow, showErrors)
}

func setStatusText(s *app.Session, in wire.SetStatusTextArgs) (wire.OKResult, error) {
	s.SetStatusText(in.Text)
	return wire.OKResult{OK: true}, nil
}

func getStatusText(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.StatusTextResult{Text: s.StatusText()})
}

func beginProgress(s *app.Session, in wire.BeginProgressArgs) (wire.BeginProgressResult, error) {
	id, err := s.Progress().Begin(in.Steps, in.Message)
	if err != nil {
		return wire.BeginProgressResult{}, err
	}
	return wire.BeginProgressResult{ID: id}, nil
}

func updateProgress(s *app.Session, in wire.UpdateProgressArgs) (wire.UpdateProgressResult, error) {
	cancelled, err := s.Progress().Update(in.ID, in.Step, in.Message)
	if err != nil {
		return wire.UpdateProgressResult{}, err
	}
	return wire.UpdateProgressResult{OK: true, Cancelled: cancelled}, nil
}

func endProgress(s *app.Session, in wire.EndProgressArgs) (wire.OKResult, error) {
	if err := s.Progress().End(in.ID); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

func registerBalloonTip(s *app.Session, in wire.RegisterBalloonTipArgs) (wire.OKResult, error) {
	spec := app.BalloonTipSpec{ID: in.ID, Title: in.Title, Text: in.Text, Icon: in.Icon}
	if err := s.BalloonTips().Register(spec); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

func showBalloonTip(s *app.Session, in wire.ShowBalloonTipArgs) (wire.ShowBalloonTipResult, error) {
	shown, err := s.ShowBalloonTip(in.ID)
	if err != nil {
		return wire.ShowBalloonTipResult{}, err
	}
	return wire.ShowBalloonTipResult{Shown: shown}, nil
}

func showPrompt(s *app.Session, in wire.ShowPromptArgs) (wire.ShowPromptResult, error) {
	resolved, answer, err := s.ShowPrompt(app.PromptSpec{
		ID: in.ID, Message: in.Message, Buttons: in.Buttons,
		Default: in.Default, Restriction: in.Restriction,
	})
	if err != nil {
		return wire.ShowPromptResult{}, err
	}
	return wire.ShowPromptResult{Resolved: resolved, Answer: answer}, nil
}

func addErrorMessage(s *app.Session, in wire.AddErrorMessageArgs) (wire.OKResult, error) {
	s.Messages().AddMessage(in.Text, in.Severity)
	return wire.OKResult{OK: true}, nil
}

func beginMessageSection(s *app.Session, in wire.BeginMessageSectionArgs) (wire.BeginMessageSectionResult, error) {
	return wire.BeginMessageSectionResult{Section: s.Messages().BeginSection(in.Title)}, nil
}

func endMessageSection(s *app.Session, in wire.EndMessageSectionArgs) (wire.OKResult, error) {
	if err := s.Messages().EndSection(in.Section); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

func listErrors(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	m := s.Messages()
	return json.Marshal(wire.ListErrorsResult{
		Root: m.View(), HasErrors: m.HasErrors(), HasWarnings: m.HasWarnings(),
		LastMessage: m.LastMessage(),
	})
}

func clearErrors(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	s.Messages().Clear()
	return ok()
}

func showErrors(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	s.SetMessageCenterOpen(true)
	return ok()
}
