// SPDX-License-Identifier: GPL-2.0-only

// Package renderer turns scene geometry into a draw list and submits it to a
// backend. Per ADR-0014 everything except raw GPU submission lives here as pure
// functions over data ("draw-call-as-data"): tessellation→draw-list building,
// culling, object-id assignment. The [NullBackend] records the submitted draw
// stream so the entire pipeline is asserted on the CPU with no device — the GPU
// "head" (a Vulkan backend) is added later behind the same [Backend] interface,
// alongside an offscreen software-Vulkan backend for image-based oracles.
package renderer
