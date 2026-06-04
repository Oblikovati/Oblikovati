// SPDX-License-Identifier: GPL-2.0-only

package envimage

import "github.com/Oblikovati/oblikovati/renderer"

// presetRes is the resolution of the generated presets. A modest lat-long map is plenty for
// IBL (the shader blurs it by roughness via mips); the skybox samples the same texture.
const presetW, presetH = 256, 128

// presetEquirect generates the built-in HDR sky for a preset. Each is a vertical gradient by
// "topness" (+1 zenith → −1 nadir), some with a sun disk; values are linear and may exceed 1.
func presetEquirect(p renderer.EnvironmentPreset) Equirect {
	switch p {
	case renderer.EnvOutdoors:
		return generate(outdoorsShade)
	case renderer.EnvOvercast:
		return generate(overcastShade)
	case renderer.EnvSunset:
		return generate(sunsetShade)
	default: // EnvStudio (and any unknown active preset)
		return generate(studioShade)
	}
}

// shadeFunc returns the linear RGB at an equirect pixel from its azimuth u∈[0,1) and topness
// t∈[-1,1] (the cosine of the polar angle, +1 at the zenith).
type shadeFunc func(u, t float32) (r, g, b float32)

// generate fills a preset-resolution equirect from a per-pixel shade function.
func generate(shade shadeFunc) Equirect {
	e := newEquirect(presetW, presetH)
	for y := 0; y < presetH; y++ {
		t := 1 - 2*(float32(y)+0.5)/float32(presetH)
		for x := 0; x < presetW; x++ {
			u := (float32(x) + 0.5) / float32(presetW)
			r, g, b := shade(u, t)
			e.set(x, y, r, g, b)
		}
	}
	return e
}

// studioShade is a soft neutral surround: bright above, mid at the sides, dimmer below.
func studioShade(_, t float32) (float32, float32, float32) {
	v := 0.35 + 0.55*smoothstep(-0.6, 0.8, t)
	return v, v, v
}

// overcastShade is a near-uniform dim grey, marginally brighter overhead.
func overcastShade(_, t float32) (float32, float32, float32) {
	v := 0.5 + 0.12*clamp01((t+1)*0.5)
	return v, v, v
}

// outdoorsShade is a sunny sky: blue zenith → pale horizon → brown ground, with a soft sun.
func outdoorsShade(u, t float32) (float32, float32, float32) {
	if t < 0 { // ground
		g := 0.16 + 0.06*(1+t)
		return g, g * 0.92, g * 0.8
	}
	r := mix(0.72, 0.26, t)
	g := mix(0.80, 0.46, t)
	b := mix(0.95, 0.88, t)
	s := sunDisk(u, t, 0.25, 0.55, 0.045, 6)
	return r + s, g + s*0.97, b + s*0.9
}

// overcastSun-free; sunsetShade is a warm low sun against a deep-blue zenith and dark ground.
func sunsetShade(u, t float32) (float32, float32, float32) {
	if t < 0 { // ground
		g := 0.05 + 0.03*(1+t)
		return g, g * 0.85, g * 0.8
	}
	r := mix(1.30, 0.12, t)
	g := mix(0.55, 0.14, t)
	b := mix(0.28, 0.38, t)
	s := sunDisk(u, t, 0.5, 0.08, 0.05, 9)
	return r + s, g + s*0.5, b + s*0.18
}

// sunDisk returns a soft additive sun highlight at azimuth su / topness st, of angular radius
// rad (in topness/azimuth units) and peak intensity peak. Azimuth wraps at 1.
func sunDisk(u, t, su, st, rad, peak float32) float32 {
	du := absf(u - su)
	if du > 0.5 {
		du = 1 - du
	}
	d2 := du*du + (t-st)*(t-st)
	g := expf(-d2 / (2 * rad * rad))
	return peak * g
}
