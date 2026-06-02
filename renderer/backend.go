// SPDX-License-Identifier: GPL-2.0-only

package renderer

// Backend submits a draw list to a rendering target. Implementations: the GPU
// "head" (Vulkan, windowed), an offscreen software-Vulkan target (image oracles),
// and [NullBackend] (records the command stream, draws nothing) for headless tests.
// Keeping submission behind this interface is what lets the whole draw pipeline be
// tested with no device (ADR-0014).
type Backend interface {
	// Render submits a frame's draw list.
	Render(DrawList)
}

// NullBackend records every submitted draw list, drawing nothing — so tests assert
// exactly what the renderer *would* draw (item count, primitives, object ids,
// colors) without a GPU.
type NullBackend struct {
	Frames []DrawList
}

// Render records the draw list as one frame.
func (n *NullBackend) Render(list DrawList) { n.Frames = append(n.Frames, list) }

// LastFrame returns the most recently submitted draw list (panics if none).
func (n *NullBackend) LastFrame() DrawList { return n.Frames[len(n.Frames)-1] }

// FrameCount returns how many frames have been submitted.
func (n *NullBackend) FrameCount() int { return len(n.Frames) }

// Reset clears recorded frames.
func (n *NullBackend) Reset() { n.Frames = nil }
