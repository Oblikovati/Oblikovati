// SPDX-License-Identifier: GPL-2.0-only

package build

// Build metadata, overridden at link time by the Makefile / goreleaser:
//
//	-ldflags "-X oblikovati.org/build.Version=v0.1.0"
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
