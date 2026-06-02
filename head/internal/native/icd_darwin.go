//go:build darwin && cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import (
	"os"
	"path/filepath"
)

// init points the Vulkan loader at the MoltenVK ICD bundled inside our .app, so a
// downloaded release runs on a clean Mac with no Vulkan SDK and no environment setup.
//
// macOS has no system Vulkan: the loader only finds a driver via VK_ICD_FILENAMES or
// system search paths that a release must not depend on. We ship MoltenVK in the
// bundle and set the variable here, in-process, BEFORE any vkCreateInstance — unlike a
// launcher's DYLD_* vars, an in-process setenv survives the hardened runtime that
// notarization requires (cgo keeps Go's os.Setenv in sync with libc getenv, which the
// loader reads). GLFW finds the loader itself via the bundle's Frameworks dir, so no
// DYLD_LIBRARY_PATH is needed.
//
// Guards keep this inert outside the bundle: an explicit override (developer machines)
// or a missing bundled manifest (plain `go run`, tests) leaves the environment alone.
func init() {
	if os.Getenv("VK_ICD_FILENAMES") != "" || os.Getenv("VK_DRIVER_FILES") != "" {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	// Bundle layout: Oblikovati.app/Contents/MacOS/<exe> with the manifest one level up
	// under Resources (see scripts/package-macos.sh).
	icd := filepath.Join(filepath.Dir(exe), "..", "Resources", "vulkan", "icd.d", "MoltenVK_icd.json")
	if _, err := os.Stat(icd); err != nil {
		return
	}
	// Ignore the error: a failure here just falls back to the loader's default search,
	// which is exactly the behavior we have without this shim.
	_ = os.Setenv("VK_ICD_FILENAMES", icd)
}
