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

	"oblikovati/head/internal/native"
)

func main() {
	frames := flag.Int("frames", 30, "number of frames to render before exiting")
	vp := flag.Bool("viewport", false, "also exercise the 3D viewport (lighting + IBL) path")
	flag.Parse()

	run := native.RunSmoke
	if *vp {
		run = native.RunViewportSmoke
	}
	if code := run(*frames); code != 0 {
		fmt.Fprintf(os.Stderr, "head smoke failed at init step %d\n", code)
		os.Exit(code)
	}
	fmt.Fprintf(os.Stdout, "head smoke ok: rendered up to %d frames\n", *frames)
}
