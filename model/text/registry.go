// SPDX-License-Identifier: GPL-2.0-only

package text

import (
	_ "embed"
	"strings"
	"sync"
)

// LiberationSans-Regular is vendored under the SIL Open Font License (see
// fonts/LICENSE-LiberationSans.txt) as the application's default sketch-text face. It is
// metric-compatible with Arial, so it doubles as the fallback for common sans families.
// Embedding it keeps text->outline resolution headless (the model never needs a host-side
// font path), which is what lets an emboss recompute its referenced text geometry.
//
//go:embed fonts/LiberationSans-Regular.ttf
var liberationSans []byte

// DefaultFontFamily is the family used when a text entity names no font.
const DefaultFontFamily = "Liberation Sans"

// FontResolver maps a font family name to a parsed Font. Wrapping resolution behind this
// interface lets the model depend on fonts without depending on the filesystem or the cgo
// head, and lets tests inject a fake catalogue (CLAUDE.md: wrap third-party I/O).
type FontResolver interface {
	// Resolve returns the font for the family; an unknown/empty family falls back to the
	// default face so text always renders.
	Resolve(family string) (*Font, error)
}

// embeddedFonts is the default FontResolver: it serves the vendored faces, falling back to
// the default for anything it does not recognise.
type embeddedFonts struct {
	once     sync.Once
	parsed   *Font
	parseErr error
}

var defaultFonts = &embeddedFonts{}

// DefaultResolver returns the process-wide resolver backed by the embedded faces.
func DefaultResolver() FontResolver { return defaultFonts }

// Resolve serves the embedded default face for every family for now (Liberation Sans is
// Arial-metric-compatible, so it stands in for the common sans families); the signature
// already takes the family so additional vendored faces can be added without a call-site
// change.
func (e *embeddedFonts) Resolve(family string) (*Font, error) {
	_ = strings.TrimSpace(family) // family is honoured once more faces are vendored
	e.once.Do(func() { e.parsed, e.parseErr = Parse(liberationSans) })
	if e.parseErr != nil {
		return nil, e.parseErr
	}
	return e.parsed, nil
}
