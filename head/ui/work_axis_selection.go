// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati/app"
	"oblikovati/model/feature"
)

func selectedWorkAxis(s *app.Session) *feature.WorkAxis {
	for _, item := range s.Selection().Items() {
		if h, ok := item.(app.WorkAxisHandle); ok {
			return h.Axis
		}
	}
	return nil
}
