// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"fmt"
	"io"

	"oblikovati/build"
)

// cmdVersion prints the build identity as plain text (CLAUDE.md: plain text for
// user-facing CLI output, JSON only for debugging logs).
func cmdVersion(out io.Writer) error {
	fmt.Fprintf(out, "oblikovati-cli %s (commit %s, built %s)\n",
		build.Version, build.Commit, build.Date)
	return nil
}
