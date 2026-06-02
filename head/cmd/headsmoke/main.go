// SPDX-License-Identifier: GPL-2.0-only

// Command headsmoke validates the native head stack end to end: it opens a real
// GLFW+Vulkan window with a Dear ImGui frame and renders a bounded number of frames.
// It exists so "does the GPU/window/ImGui stack work on this machine" is a single
// runnable check, separate from the (future) full application loop.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Oblikovati/oblikovati/head/internal/native"
)

func main() {
	frames := flag.Int("frames", 30, "number of frames to render before exiting")
	flag.Parse()

	if code := native.RunSmoke(*frames); code != 0 {
		fmt.Fprintf(os.Stderr, "head smoke failed at init step %d\n", code)
		os.Exit(code)
	}
	fmt.Printf("head smoke ok: rendered up to %d frames\n", *frames)
}
