// SPDX-License-Identifier: GPL-2.0-only

package theme

import "oblikovati/api/types"

// Palette is a full-snapshot color map: every theme stores a color for every token, so
// there is no inheritance to resolve at read time (the user chose self-contained custom
// themes — ADR-0021). Build one with [NewPalette] from a token→hex map.
type Palette map[Token]Rgba

// fallbackColor is returned for a token a palette is missing — a conspicuous magenta, so
// an incomplete palette is visible on screen rather than silently black. A complete
// palette (the built-ins, and any custom copied from one) never hits this.
var fallbackColor = Rgba{R: 1, G: 0, B: 1, A: 1}

// Color returns the token's color, or [fallbackColor] if the palette omits it.
func (p Palette) Color(t Token) Rgba {
	if c, ok := p[t]; ok {
		return c
	}
	return fallbackColor
}

// Clone returns an independent copy, so editing a custom theme never mutates the
// built-in palette it was copied from.
func (p Palette) Clone() Palette {
	out := make(Palette, len(p))
	for t, c := range p {
		out[t] = c
	}
	return out
}

// Hex renders the palette as the token→"#RRGGBBAA" map used in theme files and on the
// wire ([oblikovati/api/wire.ThemeView]).
func (p Palette) Hex() map[string]string {
	out := make(map[string]string, len(p))
	for t, c := range p {
		out[string(t)] = c.Hex()
	}
	return out
}

// NewPalette builds a palette from a token→hex map (a loaded theme file). It errors,
// naming the offending token, on any malformed hex; missing tokens are tolerated here
// and resolved to [fallbackColor] at read time, so a partial file still loads.
func NewPalette(hex map[string]string) (Palette, error) {
	out := make(Palette, len(hex))
	for k, v := range hex {
		c, err := types.ParseHex(v)
		if err != nil {
			return nil, err
		}
		out[Token(k)] = c
	}
	return out, nil
}
