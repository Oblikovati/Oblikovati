// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"sync"

	"oblikovati.org/model/text"
	"oblikovati.org/osfont"
)

// fontChoice is one selectable face in the text tool's Font dropdown: a display Label, the
// resolved Family, and — for a host font — the file Path whose bytes get embedded into the
// document on commit ("" for an application-bundled face). See ADR-0031.
type fontChoice struct {
	Label  string
	Family string
	Path   string
}

var (
	fontCatalogOnce  sync.Once
	fontCatalogCache []fontChoice
)

// fontCatalog returns the faces the text tool offers — the application's bundled faces first,
// then the host's installed fonts. Scanned once per process (OS fonts don't change mid-session;
// re-scanning every UI frame would be far too slow).
func fontCatalog() []fontChoice {
	fontCatalogOnce.Do(func() {
		for _, family := range text.EmbeddedFamilies() {
			fontCatalogCache = append(fontCatalogCache, fontChoice{Label: family, Family: family})
		}
		for _, f := range osfont.System() {
			label := f.Family
			if f.Style != "" {
				label += " — " + f.Style
			}
			fontCatalogCache = append(fontCatalogCache, fontChoice{Label: label, Family: f.Family, Path: f.Path})
		}
	})
	return fontCatalogCache
}

// fontCatalogLabels returns the dropdown labels in catalogue order.
func fontCatalogLabels() []string {
	faces := fontCatalog()
	labels := make([]string, len(faces))
	for i, f := range faces {
		labels[i] = f.Label
	}
	return labels
}
