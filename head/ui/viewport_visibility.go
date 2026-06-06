// SPDX-License-Identifier: GPL-2.0-only

package ui

import "oblikovati/app"

func shouldDrawViewport(s *app.Session) bool {
	return s != nil && s.Workspace().Count() > 0
}
