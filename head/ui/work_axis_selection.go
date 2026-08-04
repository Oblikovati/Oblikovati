// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

// selectedWorkAxis returns the first selected datum axis, or nil. It takes the selection rather
// than the whole session — the overlay only ever needed to read what is selected (audit I5).
func selectedWorkAxis(sel *app.Selection) *feature.WorkAxis {
	for _, item := range sel.Items() {
		if h, ok := item.(app.WorkAxisHandle); ok {
			return h.Axis
		}
	}
	return nil
}
