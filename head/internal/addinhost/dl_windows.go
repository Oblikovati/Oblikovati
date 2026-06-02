//go:build windows

// SPDX-License-Identifier: GPL-2.0-only

package addinhost

import "errors"

// openLibrary on Windows is a fast-follow: passing the host's //export'd callback
// to a c-shared add-in via LoadLibrary/GetProcAddress needs its own trampoline
// wiring. Until then loading errors out clearly rather than silently no-op'ing, so
// the .dll story is explicit. (Linux/macOS .so/.dylib loading is implemented in
// dl_unix.go.)
func openLibrary(path string) (addInLib, error) {
	return nil, errors.New("addinhost: shared-library add-ins are not yet supported on Windows (.dll loading is a fast-follow); path=" + path)
}
