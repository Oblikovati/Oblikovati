// SPDX-License-Identifier: GPL-2.0-only

package build

// Build metadata, overridden at link time by the Makefile / release workflows (the
// version comes from cmd/obkversion — see RELEASING.md):
//
//	-ldflags "-X oblikovati.org/build.Version=0.000200.1.0"
//
// They are vars (not consts) precisely so the linker can set them.
var (
	// Version is the semantic version of the build, or "dev" for local builds.
	Version = "dev"
	// Commit is the short git SHA the binary was built from.
	Commit = "none"
	// Date is the RFC3339 UTC build timestamp.
	Date = "unknown"
)

// AppName is the product name shown to the user (window title, About box, …).
const AppName = "Oblikovati"

// Title is the application window title: the product name plus the build version,
// e.g. "Oblikovati 0.1.0" (or "Oblikovati dev" for a local build). It is the single
// source of truth for that string, used by the GLFW window title and the
// host-reported window caption.
func Title() string { return AppName + " " + Version }
