// SPDX-License-Identifier: GPL-2.0-only

// Command oblikovati-cli is the headless entry point — batch processing,
// translation, and thumbnail export. It links no native code, so it builds and
// runs with CGO_ENABLED=0 (architecture/ADR-0008).
package main

import (
	"log/slog"
	"os"

	"github.com/Oblikovati/oblikovati/build"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	log.Info("oblikovati-cli",
		"version", build.Version,
		"commit", build.Commit,
		"date", build.Date,
		"debug", build.Debug,
	)
}
