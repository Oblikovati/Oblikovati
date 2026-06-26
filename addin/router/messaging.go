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
	r.readOnly(wire.MethodStatusSetText, setStatusText)
	r.readOnly(wire.MethodStatusGetText, getStatusText)
	r.readOnly(wire.MethodProgressBegin, beginProgress)
	r.readOnly(wire.MethodProgressUpdate, updateProgress)
	r.readOnly(wire.MethodProgressEnd, endProgress)
	r.readOnly(wire.MethodBalloonTipRegister, registerBalloonTip)
	r.readOnly(wire.MethodBalloonTipShow, showBalloonTip)
	r.readOnly(wire.MethodPromptsShow, showPrompt)
	r.readOnly(wire.MethodErrorsAddMessage, addErrorMessage)
	r.readOnly(wire.MethodErrorsBeginSection, beginMessageSection)
	r.readOnly(wire.MethodErrorsEndSection, endMessageSection)
	r.readOnly(wire.MethodErrorsList, listErrors)
	r.readOnly(wire.MethodErrorsClear, clearErrors)
	r.readOnly(wire.MethodErrorsShow, showErrors)
}

func setStatusText(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.SetStatusTextArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	s.SetStatusText(req.Text)
	return ok()
}

func getStatusText(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.StatusTextResult{Text: s.StatusText()})
}

func beginProgress(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.BeginProgressArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	id, err := s.Progress().Begin(req.Steps, req.Message)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.BeginProgressResult{ID: id})
}

func updateProgress(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.UpdateProgressArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	cancelled, err := s.Progress().Update(req.ID, req.Step, req.Message)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.UpdateProgressResult{OK: true, Cancelled: cancelled})
}

func endProgress(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.EndProgressArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.Progress().End(req.ID); err != nil {
		return nil, err
	}
	return ok()
}

func registerBalloonTip(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.RegisterBalloonTipArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	spec := app.BalloonTipSpec{ID: req.ID, Title: req.Title, Text: req.Text, Icon: req.Icon}
	if err := s.BalloonTips().Register(spec); err != nil {
		return nil, err
	}
	return ok()
}

func showBalloonTip(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.ShowBalloonTipArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	shown, err := s.ShowBalloonTip(req.ID)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.ShowBalloonTipResult{Shown: shown})
}

func showPrompt(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.ShowPromptArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	resolved, answer, err := s.ShowPrompt(app.PromptSpec{
		ID: req.ID, Message: req.Message, Buttons: req.Buttons,
		Default: req.Default, Restriction: req.Restriction,
	})
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.ShowPromptResult{Resolved: resolved, Answer: answer})
}

func addErrorMessage(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.AddErrorMessageArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	s.Messages().AddMessage(req.Text, req.Severity)
	return ok()
}

func beginMessageSection(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.BeginMessageSectionArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	return json.Marshal(wire.BeginMessageSectionResult{Section: s.Messages().BeginSection(req.Title)})
}

func endMessageSection(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.EndMessageSectionArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := s.Messages().EndSection(req.Section); err != nil {
		return nil, err
	}
	return ok()
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
