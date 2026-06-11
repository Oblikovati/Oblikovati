// SPDX-License-Identifier: GPL-2.0-only

// Package sysopen is the thin wrapper around the platform's URL opener (the
// third-party-wrapper rule). It satisfies app.URLOpener for the M05-F08 web-view
// fallback. The opener binaries are resolved to ABSOLUTE paths in fixed system
// directories — never through $PATH, which a local attacker could prepend to
// (the SonarCloud S4036 hotspot this design answers).
package sysopen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	opener, err := openerBinary()
	if err != nil {
		return err
	}
	return exec.Command(opener, openerArgs(url)...).Start()
}

// openerBinary resolves the platform opener to an absolute path in a fixed
// system directory, so the launch never consults $PATH.
func openerBinary() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return "/usr/bin/open", nil // part of the macOS base system
	case "windows":
		root := os.Getenv("SystemRoot")
		if root == "" {
			root = `C:\Windows`
		}
		return filepath.Join(root, "System32", "rundll32.exe"), nil
	default:
		return firstExisting("/usr/bin/xdg-open", "/usr/local/bin/xdg-open", "/bin/xdg-open")
	}
}

// openerArgs builds the platform-specific argv (the URL is always a single
// argument — no shell is involved).
func openerArgs(url string) []string {
	if runtime.GOOS == "windows" {
		return []string{"url.dll,FileProtocolHandler", url}
	}
	return []string{url}
}

// firstExisting returns the first candidate that exists, naming them all when
// none does (so the failure says exactly what to install).
func firstExisting(candidates ...string) (string, error) {
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c, nil
		}
	}
	return "", fmt.Errorf("sysopen: no URL opener found (looked for %s)", strings.Join(candidates, ", "))
}
