// SPDX-License-Identifier: GPL-2.0-only

package theme

import (
	_ "embed"
	"fmt"
	"sync"
)

// The shipped built-ins are Blender theme XML files embedded in the binary (ADR-0032):
// the file IS the theme — recoloring a default means editing the XML, not Go code. Each
// is parsed once; the accessors hand out independent clones so the library never shares
// mutable state between sessions or with customs duplicated from a built-in.
var (
	//go:embed dark.xml
	darkXML []byte
	//go:embed light.xml
	lightXML []byte
)

// DefaultDark is the shipped dark theme, decoded from the embedded dark.xml.
func DefaultDark() *Theme { return darkBuiltin().clone() }

// DefaultLight is the shipped light theme, decoded from the embedded light.xml.
func DefaultLight() *Theme { return lightBuiltin().clone() }

// Builtins returns the two shipped themes, dark first (the out-of-the-box default).
func Builtins() []*Theme { return []*Theme{DefaultDark(), DefaultLight()} }

var (
	darkBuiltin  = sync.OnceValue(func() *Theme { return mustBuiltin("Dark", KindDark, darkXML) })
	lightBuiltin = sync.OnceValue(func() *Theme { return mustBuiltin("Light", KindLight, lightXML) })
)

// mustBuiltin decodes one embedded theme file, panicking on failure — the built-ins are
// compiled-in assets, so a decode error is a programming/asset error caught by
// [TestDefaultsComplete], never a runtime condition.
func mustBuiltin(name string, kind Kind, data []byte) *Theme {
	t, err := decodeThemeXML(data, name, kind)
	if err != nil {
		panic(fmt.Sprintf("theme: embedded %s theme is invalid: %v", name, err))
	}
	return t
}
