// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/renderer"

// lightingViewCommands are the View-tab lighting controls (M16/F03, ADR-0026): a Lighting Style
// gallery, an Environment picker, and the shadow toggles. They mirror the Visual Style panel's
// command pattern so the ribbon stays uniform and headless-testable (the head renders the
// commands; the state lives on the Session).
func lightingViewCommands() []*CommandDefinition {
	cmds := lightingStyleCommands()
	cmds = append(cmds, environmentCommands()...)
	cmds = append(cmds, shadowCommands()...)
	return append(cmds, lightingSettingsCommand())
}

// lightingSettingsCommand is the View tab's "Lighting…" button: it toggles the Lighting
// settings panel (exposure/brightness/ambience sliders + per-light editing).
func lightingSettingsCommand() *CommandDefinition {
	return NewCommand("View.LightingSettings", "Lighting…", "Lighting Style", func(s *Session) error {
		if s.LightingPanelOpen() {
			s.CloseLightingPanel()
		} else {
			s.OpenLightingPanel()
		}
		return nil
	}).WithTab("View").WithIcon("lighting").WithButtonStyle(SmallIconButton).
		WithTooltip("Lighting settings — exposure, brightness, ambience, and per-light controls.").
		WithActive(func(s *Session) bool { return s.LightingPanelOpen() })
}

// lightingStyleCommands are the View tab's Lighting Style selection box: each preset from the
// renderer gallery as a mutually-exclusive option that activates that rig.
func lightingStyleCommands() []*CommandDefinition {
	var cmds []*CommandDefinition
	for _, opt := range renderer.LightingStyleGallery() {
		cmds = append(cmds, NewCommand("View.Lighting."+opt.Name, opt.Name, "Lighting Style",
			func(s *Session) error { return s.SetLightingStyle(opt.Name) }).
			WithTab("View").WithKind(ComboControl).
			WithTooltip("Lighting Style — "+opt.Name).
			WithActive(func(s *Session) bool { return s.LightingStyleName() == opt.Name }))
	}
	return cmds
}

// environmentCommands are the View tab's Environment selection box: each built-in HDR preset
// (plus None) as an option that sets the IBL environment and shows it as the background.
func environmentCommands() []*CommandDefinition {
	var cmds []*CommandDefinition
	for _, opt := range renderer.EnvironmentGallery() {
		cmds = append(cmds, NewCommand("View.Environment."+opt.Name, opt.Name, "Environment",
			func(s *Session) error {
				s.SetEnvironment(EnvironmentState{
					Preset: opt.Name, Intensity: 1, ShowImage: opt.Preset != renderer.EnvNone,
				})
				return nil
			}).WithTab("View").WithKind(ComboControl).
			WithTooltip("Environment — "+opt.Name).
			WithActive(func(s *Session) bool {
				e := s.Environment()
				return e.FilePath == "" && e.Preset == opt.Name
			}))
	}
	return append(cmds, NewCommand("View.LoadHDR", "Load HDR…", "Environment",
		func(s *Session) error { s.RequestLoadEnvironment(); return nil }).
		WithTab("View").WithIcon("load-hdr").WithButtonStyle(SmallIconButton).
		WithTooltip("Load HDR — use an equirectangular .hdr file as the environment.").
		WithActive(func(s *Session) bool { return s.Environment().FilePath != "" }))
}

// shadowCommands are the View tab's Shadows toggles (object/ground/ambient). Turning object
// shadows on with a zero density seeds a visible default so the toggle has an immediate effect.
func shadowCommands() []*CommandDefinition {
	return []*CommandDefinition{
		shadowToggle("View.ObjectShadows", "Object Shadows", "shadow-object",
			"Object Shadows — bodies cast shadows from the primary light.",
			func(sh *ShadowRig) { sh.ObjectShadows = !sh.ObjectShadows },
			func(sh ShadowRig) bool { return sh.ObjectShadows }),
		shadowToggle("View.GroundShadows", "Ground Shadows", "shadow-ground",
			"Ground Shadows — shadows cast onto the ground plane.",
			func(sh *ShadowRig) { sh.GroundShadows = !sh.GroundShadows },
			func(sh ShadowRig) bool { return sh.GroundShadows }),
		shadowToggle("View.AmbientShadows", "Ambient Shadows", "shadow-ambient",
			"Ambient Shadows — contact ambient occlusion.",
			func(sh *ShadowRig) { sh.AmbientShadows = !sh.AmbientShadows },
			func(sh ShadowRig) bool { return sh.AmbientShadows }),
	}
}

// shadowToggle builds one Shadows-panel toggle: flip mutates the session's shadow settings and
// checked reports the current state. Enabling any shadow with a zero density seeds a visible
// default so the toggle has an immediate effect.
func shadowToggle(id, label, icon, tip string, flip func(*ShadowRig),
	checked func(ShadowRig) bool,
) *CommandDefinition {
	return NewCommand(id, label, "Shadows", func(s *Session) error {
		sh := s.ShadowSettings()
		flip(&sh)
		if (sh.ObjectShadows || sh.GroundShadows || sh.AmbientShadows) && sh.Density == 0 {
			sh.Density, sh.Softness = 0.6, 0.4
		}
		s.SetShadowSettings(sh)
		return nil
	}).WithTab("View").WithKind(ToggleControl).WithIcon(icon).WithButtonStyle(SmallIconButton).WithTooltip(tip).
		WithActive(func(s *Session) bool { return checked(s.ShadowSettings()) })
}
