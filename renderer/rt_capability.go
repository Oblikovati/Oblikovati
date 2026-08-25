// SPDX-License-Identifier: GPL-2.0-only

package renderer

// RTDeviceFeatures is the presence-only device-extension probe for hardware ray tracing
// (M45-F01 PBI-332, ADR-0053): whether the selected Vulkan physical device advertises
// the acceleration-structure and ray-tracing-pipeline/ray-query extensions the hardware
// [Intersector] backend needs (PBI-333). It mirrors head/internal/native's
// RayTracingExtensionSupport, kept as plain data here so the decision below is
// unit-testable without a live GPU (head's cgo boundary cannot be exercised in a Go
// unit test).
type RTDeviceFeatures struct {
	AccelerationStructure bool
	RayTracingPipeline    bool
	RayQuery              bool
}

// SupportsHardwareRayTracing reports whether f is sufficient to offer the hardware-RT
// checkbox as available: acceleration structures plus either the ray-tracing pipeline
// or ray-query extension (either lets the hardware backend trace rays; PBI-333 picks
// whichever is present). This is the checkbox's default-on-iff-supported rule (ADR-0053)
// — a pure function so it is exercised by injected structs, no live GPU required.
func SupportsHardwareRayTracing(f RTDeviceFeatures) bool {
	return f.AccelerationStructure && (f.RayTracingPipeline || f.RayQuery)
}

// ResolveHardwareRayTracingEnabled turns a user override (persistence/userprefs.Prefs.
// HardwareRayTracing — nil when the user has never toggled the checkbox) and the current
// device's capability into the effective on/off state the path tracer should use:
// explicit override wins outright; otherwise it defaults to whatever the device supports
// (ADR-0053 — the checkbox starts on iff supported, and unchecking it only ever costs
// convergence time, never correctness, because the software backend is always available).
func ResolveHardwareRayTracingEnabled(override *bool, deviceSupported bool) bool {
	if override != nil {
		return *override
	}
	return deviceSupported
}
