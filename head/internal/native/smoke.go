//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import "fmt"

// RunSmoke opens the real window and renders up to maxFrames Dear ImGui frames, then
// tears everything down — proving the native stack (window, Vulkan device, swapchain,
// ImGui backends) and the Go↔C frame/widget bindings work on this machine. Returns 0
// on success, non-zero if window creation failed. maxFrames bounds the loop so the
// check cannot hang in automation.
func RunSmoke(maxFrames int) int {
	w, err := CreateWindow(1280, 720, "Oblikovati (smoke)")
	if err != nil {
		return 1
	}
	defer w.Destroy()
	for i := 0; i < maxFrames && !w.ShouldClose(); i++ {
		w.BeginFrame()
		if Begin("Oblikovati") {
			Text(fmt.Sprintf("native head smoke: frame %d", i))
		}
		End()
		w.EndFrame(0.10, 0.10, 0.12)
	}
	return 0
}
