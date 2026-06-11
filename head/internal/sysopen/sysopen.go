// SPDX-License-Identifier: GPL-2.0-only

// Package sysopen is the thin wrapper around the platform's URL opener (the
// third-party-wrapper rule): xdg-open on Linux, open on macOS, rundll32 on
// Windows. It satisfies app.URLOpener for the M05-F08 web-view fallback.
package sysopen

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// SystemOpener opens URLs with the platform's default handler.
type SystemOpener struct{}

// OpenURL launches the platform opener for url (http/https only — a shell-escape
// guard: the web-view fallback never opens arbitrary local schemes).
func (SystemOpener) OpenURL(url string) error {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("sysopen: refusing non-http(s) url %q", url)
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
