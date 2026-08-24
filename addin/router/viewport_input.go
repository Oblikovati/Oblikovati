// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
)

// Synthesised user input (wire.MethodViewportClick / wire.MethodViewportKey): drive an interactive
// command the way a person does.
//
// Every other method on this router edits the model. That cannot reach the behaviour that only
// exists in the input path — the constraints a tool infers from where the click landed, what it
// previews between clicks, how a multi-click chain accumulates — so a client could neither
// automate nor test it. Two bugs in exactly that path (#2032) had no way to be reproduced over the
// API, which is why this exists.

// clickViewport delivers a pointer click at a viewport pixel, or at a model point the host
// projects. It returns the pixel clicked and the command still running afterwards.
func clickViewport(s *app.Session, in wire.ClickViewportArgs) (wire.ClickViewportResult, error) {
	x, y, err := clickPixel(s, in)
	if err != nil {
		return wire.ClickViewportResult{}, err
	}
	button, err := app.PointerButtonNamed(in.Button)
	if err != nil {
		return wire.ClickViewportResult{}, err
	}
	tool := s.ClickPointer(x, y, button, app.ModifierFor(in.Shift, in.Ctrl, in.Alt))
	return wire.ClickViewportResult{X: x, Y: y, ActiveTool: tool}, nil
}

// clickPixel resolves the request's position to a viewport pixel: a model point is projected
// through the live camera, otherwise the given pixel is used as-is.
func clickPixel(s *app.Session, in wire.ClickViewportArgs) (x, y float64, err error) {
	if in.Point == nil {
		return in.X, in.Y, nil
	}
	p := math.P3(in.Point.X, in.Point.Y, in.Point.Z)
	px, py, ok := s.ProjectToViewport(p)
	if !ok {
		return 0, 0, fmt.Errorf("%s: the point (%v,%v,%v) is not in view — orbit or zoom so it is on screen, or pass viewport pixels instead",
			wire.MethodViewportClick, in.Point.X, in.Point.Y, in.Point.Z)
	}
	return px, py, nil
}

// pressKey delivers a key to the running command — Escape/Enter finish a variable-length one.
func pressKey(s *app.Session, in wire.PressKeyArgs) (wire.PressKeyResult, error) {
	tool, err := s.PressKeyNamed(in.Key, app.ModifierFor(in.Shift, in.Ctrl, in.Alt))
	if err != nil {
		return wire.PressKeyResult{}, err
	}
	return wire.PressKeyResult{ActiveTool: tool}, nil
}

// scrollViewport delivers a synthetic mouse-wheel tick, zooming the camera exactly as a
// real wheel notch would (Session.ScrollViewport).
func scrollViewport(s *app.Session, in wire.ScrollViewportArgs) (wire.ScrollViewportResult, error) {
	tool := s.ScrollViewport(in.DX, in.DY, in.X, in.Y)
	return wire.ScrollViewportResult{ActiveTool: tool}, nil
}
