// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	"errors"
	"testing"

	"oblikovati.org/model/feature"
	"oblikovati.org/model/health"
)

// okFeature recomputes cleanly (a healthy feature); failingFeature always errors (goes Sick).
type okFeature struct{}

func (okFeature) Kind() string                                    { return "ok" }
func (okFeature) Recompute(feature.Input) (feature.Output, error) { return feature.Output{}, nil }

type failingFeature struct{}

func (failingFeature) Kind() string { return "failer" }
func (failingFeature) Recompute(feature.Input) (feature.Output, error) {
	return feature.Output{}, errors.New("boom")
}

// TestModelHealthOf checks the aggregation across a clean, a failing and a suppressed feature.
func TestModelHealthOf(t *testing.T) {
	fs := feature.NewPartFeatures(nil)
	fs.Add(okFeature{})
	bad := fs.Add(failingFeature{})
	bad.SetName("Extrusion2")
	supp := fs.Add(okFeature{})
	supp.SetName("Fillet1")
	supp.SetSuppressed(true)
	fs.Recompute()

	mh := ModelHealthOf(fs)
	if mh.Overall != health.Sick {
		t.Errorf("overall = %v, want sick (a feature failed)", mh.Overall)
	}
	if mh.SickCount != 1 {
		t.Errorf("sick count = %d, want 1", mh.SickCount)
	}
	if len(mh.Unhealthy) != 2 {
		t.Fatalf("unhealthy = %+v, want 2 (the sick + the suppressed)", mh.Unhealthy)
	}
	if !hasStatus(mh.Unhealthy, "Extrusion2", health.Sick) {
		t.Error("the failing feature is not reported sick")
	}
	if !hasStatus(mh.Unhealthy, "Fillet1", health.Suppressed) {
		t.Error("the suppressed feature is not reported suppressed")
	}
}

// TestModelHealthOfAllHealthy checks a clean part reports OK with nothing to repair.
func TestModelHealthOfAllHealthy(t *testing.T) {
	fs := feature.NewPartFeatures(nil)
	fs.Add(okFeature{})
	fs.Recompute()
	mh := ModelHealthOf(fs)
	if mh.Overall != health.OK || mh.SickCount != 0 || len(mh.Unhealthy) != 0 {
		t.Errorf("clean part health = %+v, want OK with no unhealthy features", mh)
	}
}

// TestHealthSeverityOrder checks the roll-up ordering: OK < Warning < Sick, Suppressed neutral.
func TestHealthSeverityOrder(t *testing.T) {
	if severity(health.OK) >= severity(health.Warning) || severity(health.Warning) >= severity(health.Sick) {
		t.Error("severity must order OK < Warning < Sick")
	}
	if severity(health.Suppressed) != severity(health.OK) {
		t.Error("suppressed must be neutral (severity of OK)")
	}
}

func hasStatus(items []FeatureHealthItem, name string, status health.Status) bool {
	for _, it := range items {
		if it.Name == name && it.Status == status {
			return true
		}
	}
	return false
}
