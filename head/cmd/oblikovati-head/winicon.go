//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package main

// The Windows executable file icon (shown by Explorer / shortcuts) is embedded via the
// committed oblikovati_windows_amd64.syso — a COFF resource object the Go linker bundles
// into the .exe. Its GOOS_GOARCH filename suffix scopes it to windows/amd64; every other
// platform ignores it and the running window/taskbar icon comes from native.SetIcon.
//
// The .syso is generated from the source SVG (head/internal/appicon/oblikovati.svg).
// Regenerate it after the SVG changes by running `go generate ./cmd/oblikovati-head`
// from the head module (rsrc is pinned so the resource is reproducible):
//
//go:generate sh -c "go run ../genappicon -format ico -out obk-appicon.ico && go run github.com/akavel/rsrc@v0.10.2 -ico obk-appicon.ico -arch amd64 -o oblikovati_windows_amd64.syso && rm -f obk-appicon.ico"
