// SPDX-License-Identifier: GPL-2.0-only

package icon

import "image"

// RoleColors is the theme's color per role (straight-alpha RGBA, channels in [0,1]),
// indexed by [Role] — the head fills it from the icon.* theme tokens.
type RoleColors [RoleCount][4]float32

// Compose layers the role masks into one straight-alpha RGBA image: background first,
// then tertiary, secondary, and primary on top (src-over). The result is what the head
// uploads as the icon's texture; ImGui draws it with a white (identity) tint, so all
// theming happens here.
func (m *RoleMasks) Compose(colors RoleColors) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, m.px, m.px))
	for i := 0; i < m.px*m.px; i++ {
		r, g, b, a := m.composePixel(i, colors)
		writeStraight(out.Pix[i*4:i*4+4], r, g, b, a)
	}
	return out
}

// composePixel accumulates the roles' premultiplied src-over at one pixel.
func (m *RoleMasks) composePixel(i int, colors RoleColors) (r, g, b, a float32) {
	for role := Role(0); role < RoleCount; role++ {
		sa := float32(m.cover[role][i]) / 255 * colors[role][3]
		if sa == 0 {
			continue
		}
		c := colors[role]
		r = c[0]*sa + r*(1-sa)
		g = c[1]*sa + g*(1-sa)
		b = c[2]*sa + b*(1-sa)
		a = sa + a*(1-sa)
	}
	return r, g, b, a
}

// writeStraight stores a premultiplied accumulation as straight-alpha bytes (ImGui
// blends with straight source alpha, so the color channels must be un-premultiplied).
func writeStraight(px []byte, r, g, b, a float32) {
	if a <= 0 {
		return // leave fully transparent black
	}
	px[0] = to255(r / a)
	px[1] = to255(g / a)
	px[2] = to255(b / a)
	px[3] = to255(a)
}

// to255 quantizes a [0,1] channel to a byte, clamping out-of-range inputs.
func to255(v float32) byte {
	switch {
	case v <= 0:
		return 0
	case v >= 1:
		return 255
	default:
		return byte(v*255 + 0.5)
	}
}
