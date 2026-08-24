// SPDX-License-Identifier: GPL-2.0-only

package renderer

import "testing"

func TestResolveHardwareRayTracingEnabled(t *testing.T) {
	trueVal, falseVal := true, false
	cases := []struct {
		name            string
		override        *bool
		deviceSupported bool
		want            bool
	}{
		{"no override, supported", nil, true, true},
		{"no override, unsupported", nil, false, false},
		{"forced on, unsupported device", &trueVal, false, true},
		{"forced off, supported device", &falseVal, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ResolveHardwareRayTracingEnabled(c.override, c.deviceSupported); got != c.want {
				t.Errorf("ResolveHardwareRayTracingEnabled(%v, %v) = %v, want %v",
					c.override, c.deviceSupported, got, c.want)
			}
		})
	}
}

func TestSupportsHardwareRayTracing(t *testing.T) {
	cases := []struct {
		name string
		f    RTDeviceFeatures
		want bool
	}{
		{"none", RTDeviceFeatures{}, false},
		{"accel struct only", RTDeviceFeatures{AccelerationStructure: true}, false},
		{"pipeline only", RTDeviceFeatures{RayTracingPipeline: true}, false},
		{"ray query only", RTDeviceFeatures{RayQuery: true}, false},
		{"accel + pipeline", RTDeviceFeatures{AccelerationStructure: true, RayTracingPipeline: true}, true},
		{"accel + ray query", RTDeviceFeatures{AccelerationStructure: true, RayQuery: true}, true},
		{"accel + both", RTDeviceFeatures{AccelerationStructure: true, RayTracingPipeline: true, RayQuery: true}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SupportsHardwareRayTracing(c.f); got != c.want {
				t.Errorf("SupportsHardwareRayTracing(%+v) = %v, want %v", c.f, got, c.want)
			}
		})
	}
}
