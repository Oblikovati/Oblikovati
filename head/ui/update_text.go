// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"

	"oblikovati.org/update"
)

// updateHeadline is the one-line status shown at the top of the Software Update window.
// It is the pure text half of the window (no native calls) so it stays unit-testable
// without the cgo/Vulkan head.
func updateHeadline(res *update.Result) string {
	if res.Skipped {
		return "Update check skipped: " + skipHeadline(res)
	}
	if res.UpdateAvailable {
		return fmt.Sprintf("A new %s release is available: %s", res.Channel, res.Latest.Version)
	}
	return fmt.Sprintf("You are up to date (%s %s).", res.Channel, res.Current)
}

// skipHeadline expands a skip reason into a sentence for the window.
func skipHeadline(res *update.Result) string {
	switch res.SkipReason {
	case "offline":
		return "no internet connection."
	case "development build":
		return "this is a development build."
	case "no published release":
		return "no release published on the " + res.Channel.String() + " channel yet."
	case "check failed":
		return "could not reach the update server."
	default:
		return res.SkipReason + "."
	}
}
