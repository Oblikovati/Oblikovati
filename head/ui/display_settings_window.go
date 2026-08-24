//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/display"
)

// drawDisplaySettingsWindow renders the Display Settings dialog while it is open (M16-F07
// #643): the active document's edge color, ground-plane visibility/color, shadow toggles and
// textures. Each edit writes the whole settings value back through the session (the same
// surface the API uses and the .obk persists).
func drawDisplaySettingsBody(s *app.Session) {
	set := s.DisplaySettings(0)
	if editDisplaySettings(&set) {
		s.SetDisplaySettings(0, set)
	}
	native.Separator()
	editRealisticSettings(s)
	native.Separator()
	if native.Button("Done") {
		s.CloseDisplaySettings()
	}
}

// editRealisticSettings draws the hardware ray-tracing checkbox (M45-F05 PBI-350,
// ADR-0053): a persisted, global (not per-document) preference, since it reflects a
// device capability trade-off rather than a document's authored look. Unchecking it
// only costs Realistic mode's convergence speed — the software backend is always
// spec-equivalent — so the label says so rather than implying a quality loss.
//
// This dialog has no live Vulkan device handle, so the checkbox shows the raw override
// (unchecked/unset until the user picks a side) — the actual effective backend,
// resolved against this device's real capability
// (renderer.ResolveHardwareRayTracingEnabled), is decided per-frame in
// realistic_render.go and shown live via the convergence indicator's rendering, not
// this checkbox's initial state.
func editRealisticSettings(s realisticSession) {
	prefs := s.ViewCubePrefs()
	enabled := prefs.HardwareRayTracing != nil && *prefs.HardwareRayTracing
	if native.Checkbox("Hardware ray tracing (Realistic mode)", &enabled) {
		prefs.HardwareRayTracing = &enabled
		s.SetViewCubePrefs(prefs)
	}
	native.TextWrapped("When off, Realistic mode still renders the exact same image — only slower to converge. Unset defaults to whatever this device supports.")
}

// editDisplaySettings draws the editable controls into set, returning whether any changed.
func editDisplaySettings(set *display.Settings) bool {
	colors := editDisplayColors(set)   // both must run so every control draws
	toggles := editDisplayToggles(set) // (no short-circuit)
	return colors || toggles
}

// editDisplayColors draws the edge / ground color editors and the ground-plane toggle.
func editDisplayColors(set *display.Settings) bool {
	changed := false
	edge := set.EdgeColor.Rgba().Array()
	if native.ColorEdit4("Edge color", &edge) {
		set.EdgeColor = colorFromVec(edge)
		changed = true
	}
	gv := set.GroundPlane.Visible
	if native.Checkbox("Ground plane", &gv) {
		set.GroundPlane.Visible = gv
		changed = true
	}
	gc := set.GroundPlane.Color.Rgba().Array()
	if native.ColorEdit4("Ground color", &gc) {
		set.GroundPlane.Color = colorFromVec(gc)
		changed = true
	}
	return changed
}

// editDisplayToggles draws the shadow / texture toggles.
func editDisplayToggles(set *display.Settings) bool {
	changed := false
	if obj := set.ShowObjectShadows; native.Checkbox("Object shadows", &obj) {
		set.ShowObjectShadows = obj
		changed = true
	}
	if refl := set.ShowGroundReflections; native.Checkbox("Ground reflections", &refl) {
		set.ShowGroundReflections = refl
		changed = true
	}
	if tex := set.TexturesOn; native.Checkbox("Textures", &tex) {
		set.TexturesOn = tex
		changed = true
	}
	return changed
}

// colorFromVec builds an opaque-override color from an ImGui rgba float vector.
func colorFromVec(v [4]float32) types.Color {
	c := types.NewColor(toByte(v[0]), toByte(v[1]), toByte(v[2]))
	c.Opacity = float64(v[3])
	return c
}

// toByte clamps a 0..1 float channel to an 8-bit value.
func toByte(f float32) uint8 {
	switch {
	case f <= 0:
		return 0
	case f >= 1:
		return 255
	default:
		return uint8(f*255 + 0.5)
	}
}
