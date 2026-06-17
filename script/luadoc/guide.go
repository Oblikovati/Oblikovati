// SPDX-License-Identifier: GPL-2.0-only

package luadoc

import _ "embed"

// guideText is the hand-written introduction that precedes the generated reference (what
// scripting is, how to run a script, the sandbox, the call forms, error handling). It lives in
// guide.md so it can use Markdown fences and inline code freely; the reference below it is
// generated, so this prose stays small and stable.
//
//go:embed guide.md
var guideText string

// guide returns the introduction Markdown.
func guide() string { return guideText }
