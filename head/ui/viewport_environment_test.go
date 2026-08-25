//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/renderer"
)

// TestEnvCacheResolveInactiveOnNoPreset and TestEnvCacheResolveActiveBuildsDistribution
// cover envCache.resolve's two branches directly (#2135/#2155's dist field/build) — pure
// image decode/generation, no native window needed.
func TestEnvCacheResolveInactiveOnNoPreset(t *testing.T) {
	var e envCache
	e.resolve(renderer.Environment{Preset: renderer.EnvNone})
	if e.active || e.dist != nil {
		t.Errorf("resolve(EnvNone): active=%v dist=%v, want inactive with a nil distribution", e.active, e.dist)
	}
}

func TestEnvCacheResolveActiveBuildsDistribution(t *testing.T) {
	var e envCache
	e.resolve(renderer.Environment{Preset: renderer.EnvSky, Intensity: 1, ShowImage: true})
	if !e.active || e.dist == nil {
		t.Errorf("resolve(EnvSky): active=%v dist=%v, want active with a built distribution", e.active, e.dist)
	}
}

// TestCurrentEnvironmentDistributionNilWhenInactive/AndCached cover
// currentEnvironmentDistribution's two branches directly — no native window needed, since
// it only reads the package-level viewportEnv cache. Each test saves/restores viewportEnv
// around a direct write, so it can't leak state into other tests sharing this process
// (viewport_environment.go's own doc comment: "the head has one 3D viewport").
func TestCurrentEnvironmentDistributionNilWhenInactive(t *testing.T) {
	saved := viewportEnv
	defer func() { viewportEnv = saved }()
	viewportEnv = envCache{active: false}

	if got := currentEnvironmentDistribution(); got != nil {
		t.Errorf("currentEnvironmentDistribution() = %v, want nil when inactive", got)
	}
}

func TestCurrentEnvironmentDistributionReturnsCachedDistWhenActive(t *testing.T) {
	saved := viewportEnv
	defer func() { viewportEnv = saved }()
	dist := solidEnvDistribution()
	viewportEnv = envCache{active: true, dist: dist}

	if got := currentEnvironmentDistribution(); got != dist {
		t.Errorf("currentEnvironmentDistribution() = %p, want %p (the cached distribution)", got, dist)
	}
}
