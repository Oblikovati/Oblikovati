// SPDX-License-Identifier: GPL-2.0-only

// Command oblikovati is the windowed application entry point. It will assemble a
// Runtime (window + Vulkan viewport); for now it only reports build metadata so
// the build/release pipeline has a real target to produce.
package main

import (
	"log/slog"
	"os"

	"oblikovati.org/build"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	log.Info("oblikovati starting", "version", build.Version, "commit", build.Commit)

	if err := build.NotYetImplemented("PBI-001: runtime/window bootstrap"); err != nil {
		log.Warn("startup incomplete", "reason", err)
	}
}
