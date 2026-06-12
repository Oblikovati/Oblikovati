// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/app/options"
)

// registerOptionHandlers wires the typed application-option groups (M05-F11, #618).
func (r *Router) registerOptionHandlers() {
	r.handlers[wire.MethodOptionsListGroups] = listOptionGroups
	r.handlers[wire.MethodOptionsGetGroup] = getOptionGroup
	r.handlers[wire.MethodOptionsSetGroup] = setOptionGroup
}

// optionGroupNames is the stable group order of options.listGroups.
var optionGroupNames = []string{
	wire.OptionGroupGeneral, wire.OptionGroupDisplay, wire.OptionGroupSketch, wire.OptionGroupPart,
	wire.OptionGroupSave,
}

// listOptionGroups returns the available group names (wire options.listGroups).
func listOptionGroups(_ *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.ListOptionGroupsResult{Groups: optionGroupNames})
}

// getOptionGroup returns one group's current values (wire options.getGroup).
func getOptionGroup(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.GetOptionGroupArgs
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	view, err := optionGroupView(s, req.Group)
	if err != nil {
		return nil, err
	}
	return json.Marshal(view)
}

// optionGroupView assembles the union view for one group. The display group proxies
// the userprefs store and the theme system — their persistence stays where it is.
func optionGroupView(s *app.Session, group string) (wire.OptionGroupView, error) {
	view := wire.OptionGroupView{Group: group}
	switch group {
	case wire.OptionGroupGeneral:
		view.General = &wire.GeneralOptionsView{StartupAction: s.Options().General.StartupAction}
	case wire.OptionGroupDisplay:
		view.Display = displayOptionsView(s)
	case wire.OptionGroupSketch:
		o := s.Options().Sketch
		view.Sketch = &wire.SketchOptionsView{
			GridSpacingCm: o.GridSpacingCm, GridVisible: o.GridVisible,
			GridMajorEvery: o.GridMajorEvery, SnapToPoints: o.SnapToPoints, SnapToGrid: o.SnapToGrid,
		}
	case wire.OptionGroupPart:
		view.Part = &wire.PartOptionsView{ChamferFlatCorners: s.Options().Part.ChamferFlatCorners}
	case wire.OptionGroupSave:
		o := s.Options().Save
		view.Save = &wire.SaveOptionsView{
			Thumbnail: o.Thumbnail, SaveDependents: o.SaveDependents, OldVersionsToKeep: o.OldVersionsToKeep,
		}
	default:
		return view, fmt.Errorf("unknown option group %q (one of general/display/sketch/part/save)", group)
	}
	return view, nil
}

// displayOptionsView assembles the display group from its proxied stores.
func displayOptionsView(s *app.Session) *wire.DisplayOptionsView {
	p := s.ViewCubePrefs()
	return &wire.DisplayOptionsView{
		ColorScheme:         s.Themes().ActiveName(),
		ViewCubeHidden:      p.CubeHidden,
		CompassHidden:       p.CompassHidden,
		LockToSelection:     p.LockToSelection,
		CubeInactiveOpacity: p.InactiveOpacity,
		CubeSizePx:          p.CubeSizePx,
		CubeCorner:          p.CubeCorner,
	}
}

// setOptionGroup writes one group (wire options.setGroup): the request names the
// group and carries exactly that group's payload.
func setOptionGroup(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var req wire.OptionGroupView
	if err := decode(args, &req); err != nil {
		return nil, err
	}
	if err := applyOptionGroup(s, req); err != nil {
		return nil, err
	}
	return ok()
}

// applyOptionGroup dispatches one group write to the session.
func applyOptionGroup(s *app.Session, req wire.OptionGroupView) error {
	switch {
	case req.Group == wire.OptionGroupGeneral && req.General != nil:
		return s.SetGeneralOptions(options.General{StartupAction: req.General.StartupAction})
	case req.Group == wire.OptionGroupDisplay && req.Display != nil:
		return applyDisplayOptions(s, *req.Display)
	case req.Group == wire.OptionGroupSketch && req.Sketch != nil:
		return s.SetSketchOptions(options.Sketch{
			GridSpacingCm: req.Sketch.GridSpacingCm, GridVisible: req.Sketch.GridVisible,
			GridMajorEvery: req.Sketch.GridMajorEvery, SnapToPoints: req.Sketch.SnapToPoints,
			SnapToGrid: req.Sketch.SnapToGrid,
		})
	case req.Group == wire.OptionGroupPart && req.Part != nil:
		return s.SetPartOptions(options.Part{ChamferFlatCorners: req.Part.ChamferFlatCorners})
	case req.Group == wire.OptionGroupSave && req.Save != nil:
		return s.SetSaveOptions(options.Save{
			Thumbnail:         req.Save.Thumbnail,
			SaveDependents:    req.Save.SaveDependents,
			OldVersionsToKeep: req.Save.OldVersionsToKeep,
		})
	default:
		return fmt.Errorf("option group %q carries no matching payload (set exactly the named group's field)", req.Group)
	}
}

// applyDisplayOptions writes the display proxies: the theme by name, the ViewCube
// preferences through their own store.
func applyDisplayOptions(s *app.Session, v wire.DisplayOptionsView) error {
	if v.ColorScheme != "" && v.ColorScheme != s.Themes().ActiveName() {
		// SetActiveTheme treats an unknown name as a no-op; the wire surface must
		// reject it instead, so a typo'd scheme is loud.
		if !themeExists(s, v.ColorScheme) {
			return fmt.Errorf("unknown color scheme %q (an existing theme name)", v.ColorScheme)
		}
		if err := s.SetActiveTheme(v.ColorScheme); err != nil {
			return err
		}
	}
	p := s.ViewCubePrefs()
	p.CubeHidden = v.ViewCubeHidden
	p.CompassHidden = v.CompassHidden
	p.LockToSelection = v.LockToSelection
	p.InactiveOpacity = v.CubeInactiveOpacity
	p.CubeSizePx = v.CubeSizePx
	p.CubeCorner = v.CubeCorner
	s.SetViewCubePrefs(p)
	return nil
}

// themeExists reports whether a theme of that name is in the library.
func themeExists(s *app.Session, name string) bool {
	for _, t := range s.Themes().Themes() {
		if t.Name() == name {
			return true
		}
	}
	return false
}
