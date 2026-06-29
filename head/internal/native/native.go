//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Package native is the thin cgo boundary to our vendored Dear ImGui + a hand-rolled
// GLFW/Vulkan renderer. It owns ALL C/C++ interop for the head (ADR-0008 keeps the
// cgo surface small and in one place); the rest of the head is ordinary Go that calls
// these wrappers. The vendored ImGui TUs are compiled via imgui_unity.cpp; the GLFW
// and Vulkan backends are separate TUs (impl_glfw.cpp / impl_vulkan.cpp).
package native

/*
#cgo CXXFLAGS: -std=c++17
#cgo CPPFLAGS: -I${SRCDIR}/../../third_party/imgui -I${SRCDIR}/../../third_party/imgui/backends
#cgo pkg-config: glfw3 vulkan

#include <stdlib.h>

void* obk_head_create(int width, int height, const char* title);
int   obk_head_should_close(void* h);
void  obk_head_set_should_close(void* h, int v);
void  obk_head_begin_frame(void* h);
void  obk_head_end_frame(void* h, float r, float g, float b);
void  obk_head_wait_events_timeout(double seconds);
void  obk_head_wait_events(void);
void  obk_head_post_empty_event(void);
void  obk_head_destroy(void* h);
void  obk_head_get_window_state(void* h, int* x, int* y, int* w, int* hh, int* maximized);
void  obk_head_apply_window_state(void* h, int x, int y, int maximized);
void  obk_head_gpu_info(void* h, char* nameOut, int nameCap, unsigned int* apiVersionOut);
void  obk_head_set_icon(void* h, int count, const int* sizes, const unsigned char* const* pixels);
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

// Window is a live native window: one GLFW window + Vulkan device/swapchain + a Dear
// ImGui context, all behind the opaque handle. Create it, drive it frame-by-frame
// (BeginFrame → build chrome via the imgui wrappers → EndFrame), then Destroy it.
type Window struct {
	handle unsafe.Pointer
}

// CreateWindow opens the native window and brings up Vulkan + ImGui. It returns an
// error if any init step (GLFW, instance, device, swapchain, ImGui) fails.
func CreateWindow(width, height int, title string) (*Window, error) {
	ctitle := C.CString(title)
	defer C.free(unsafe.Pointer(ctitle))
	h := C.obk_head_create(C.int(width), C.int(height), ctitle)
	if h == nil {
		return nil, errors.New("native: head window/Vulkan/ImGui init failed (no display or no Vulkan driver?)")
	}
	w := &Window{handle: h}
	w.SetUIFont(uiFontSizePx)     // embedded Helvetica-metric UI font (before the first frame)
	w.SetMonoFont(monoFontSizePx) // editor fixed-width face, added after the UI font clears the atlas
	return w, nil
}

// WindowState returns the window's current placement: position (virtual-screen
// coordinates, which encode the monitor), size, and whether it is maximized — for saving
// across sessions.
func (w *Window) WindowState() (x, y, width, height int, maximized bool) {
	var cx, cy, cw, ch, cm C.int
	C.obk_head_get_window_state(w.handle, &cx, &cy, &cw, &ch, &cm)
	return int(cx), int(cy), int(cw), int(ch), cm != 0
}

// ApplyWindowState restores a saved placement: move to (x,y) (same monitor) and maximize
// if it was. Size is set at CreateWindow.
func (w *Window) ApplyWindowState(x, y int, maximized bool) {
	m := C.int(0)
	if maximized {
		m = 1
	}
	C.obk_head_apply_window_state(w.handle, C.int(x), C.int(y), m)
}

// ShouldClose reports whether the user asked to close the window.
func (w *Window) ShouldClose() bool { return C.obk_head_should_close(w.handle) != 0 }

// SetShouldClose overrides the close flag: false cancels a pending close (e.g. while a
// "save changes?" prompt is up), true confirms one.
func (w *Window) SetShouldClose(v bool) {
	n := C.int(0)
	if v {
		n = 1
	}
	C.obk_head_set_should_close(w.handle, n)
}

// BeginFrame pumps window events, recreates the swapchain if needed, and starts a new
// Dear ImGui frame. Build the chrome after this, then call EndFrame.
func (w *Window) BeginFrame() { C.obk_head_begin_frame(w.handle) }

// EndFrame renders the ImGui draw data into the swapchain (clearing to the given
// background color) and presents it.
func (w *Window) EndFrame(r, g, b float32) {
	C.obk_head_end_frame(w.handle, C.float(r), C.float(g), C.float(b))
}

// WaitEvents blocks until a window event arrives or `seconds` elapse, then returns. The loop
// uses it as the short tick WHILE the UI is animating (a camera tween, an active drag, the
// burst of frames after input), so transitions stay smooth without a 60 Hz CPU-bound spin.
// Input wakes it immediately. Headless/smoke runs must not call it (they'd block per frame).
func (w *Window) WaitEvents(seconds float64) {
	C.obk_head_wait_events_timeout(C.double(seconds))
}

// WaitEventsBlocking blocks indefinitely until a window event arrives or PostEmptyEvent is
// posted, then returns. The loop uses it when the UI is fully idle: with FIFO present the
// only throttle is the present, and on a software Vulkan rasterizer (a VM's llvmpipe) every
// present rasterizes the whole frame on the CPU, so an unchanging scene pegs the cores for
// nothing (#1493). Blocking here drops idle CPU to ~0 until the user acts or a background
// producer posts a wake. Headless/smoke runs must not call it (the loop would hang).
func (w *Window) WaitEventsBlocking() {
	C.obk_head_wait_events()
}

// PostEmptyEvent wakes a loop blocked in WaitEventsBlocking from ANY goroutine (it is one of
// the few thread-safe GLFW calls). A background producer that changes what should be on
// screen without an OS input event — an add-in submitting model work, a finished update
// check — posts this so the idle loop renders the change promptly (#1493).
func PostEmptyEvent() { C.obk_head_post_empty_event() }

// GPUInfo reports the selected Vulkan physical device's name and API version (formatted
// "major.minor.patch"), for anonymous installation telemetry (#1182). On macOS the device
// runs over MoltenVK, so the version is the MoltenVK-backed Vulkan version. Both are "" when
// no device is available (e.g. a software-only or torn-down window).
func (w *Window) GPUInfo() (name, vulkanVersion string) {
	var buf [256]C.char
	var apiVer C.uint
	C.obk_head_gpu_info(w.handle, &buf[0], C.int(len(buf)), &apiVer)
	return C.GoString(&buf[0]), formatVulkanVersion(uint32(apiVer))
}

// formatVulkanVersion renders a packed VkPhysicalDeviceProperties.apiVersion as
// "major.minor.patch" (the Vulkan version bit layout), or "" for a zero (no device) value.
func formatVulkanVersion(v uint32) string {
	if v == 0 {
		return ""
	}
	return fmt.Sprintf("%d.%d.%d", (v>>22)&0x7f, (v>>12)&0x3ff, v&0xfff)
}

// Destroy tears down ImGui, the Vulkan device, and the window.
func (w *Window) Destroy() {
	C.obk_head_destroy(w.handle)
	w.handle = nil
}
