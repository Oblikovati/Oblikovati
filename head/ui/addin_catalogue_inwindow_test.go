//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// In-window verification of the Add-In Catalogue window: open the real GLFW+Vulkan+ImGui
// window and run the production draw path for a few frames against a fake catalogue, so the
// header/tabs/cards/actions render code is exercised (and covered) without a live service.
// Skips cleanly where no display/Vulkan is available.
package ui

import (
	"context"
	"testing"
	"time"

	"oblikovati.org/addincat"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// fakeCatSource serves a one-add-in catalogue for the host's API version, no network.
type fakeCatSource struct{}

func (fakeCatSource) List(_ context.Context, major, minor int, _ string) ([]addincat.Entry, error) {
	return []addincat.Entry{{
		Name: "com.oblikovati.cam", DisplayName: "Oblikovati CAM", Description: "machining", License: "Apache-2.0",
		Versions: []addincat.Version{{
			Version: "0.6.0", APIMajor: major, APIMinor: minor,
			Bundles: map[string]addincat.Bundle{addincat.Platform(): {URL: "u"}},
		}},
	}}, nil
}

// fakeCatInstaller reports every catalogue entry as available.
type fakeCatInstaller struct{}

func (fakeCatInstaller) Install(context.Context, addincat.Entry, addincat.Version) error { return nil }
func (fakeCatInstaller) Uninstall(string) error                                          { return nil }
func (fakeCatInstaller) Status(catalogue []addincat.Entry, _, _ int) ([]addincat.AddInStatus, error) {
	out := make([]addincat.AddInStatus, len(catalogue))
	for i, e := range catalogue {
		out[i] = addincat.AddInStatus{Entry: e, State: addincat.StateAvailable, LatestVersion: e.Versions[0].Version}
	}
	return out, nil
}

func waitCatalogue(t *testing.T, s *app.Session) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(s.AddInStatuses()) > 0 {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("catalogue did not populate")
}

func TestAddInCatalogueWindowDrawsInWindow(t *testing.T) {
	win, err := native.CreateWindow(900, 640, "obk-addincat-test")
	if err != nil {
		t.Skipf("no window/Vulkan available: %v", err)
	}
	defer win.Destroy()
	win.InitViewport()

	s := app.NewSession()
	s.SetAddInCatalogue(fakeCatSource{}, fakeCatInstaller{})
	OpenAddInCatalogue(s)
	waitCatalogue(t, s)

	// Cards whose state the default (first) tab does not show — drawn into a scratch window so
	// the Installed/Update title + action branches are exercised too.
	installed := addincat.AddInStatus{
		Entry: addincat.Entry{Name: "com.x.installed", DisplayName: "Installed One", Description: "d", License: "L"},
		State: addincat.StateInstalled, InstalledVersion: "1.0.0",
	}
	updatable := addincat.AddInStatus{
		Entry: addincat.Entry{Name: "com.x.update", DisplayName: "Update One"},
		State: addincat.StateUpdateAvailable, InstalledVersion: "1.0.0", LatestVersion: "1.1.0",
	}

	for i := 0; i < 3; i++ {
		win.BeginFrame()
		drawAddInCatalogueWindow(s)
		if native.Begin("##addincat-scratch") {
			drawCatalogueCard(s, installed)
			drawCatalogueCard(s, updatable)
		}
		native.End()
		win.EndFrame(0.1, 0.1, 0.1)
	}

	// Render a session whose refresh failed, so the header's error line is exercised too.
	errSession := app.NewSession()
	errSession.SetAddInCatalogue(errSource{}, fakeCatInstaller{})
	errSession.RefreshAddInCatalogue("")
	deadline := time.Now().Add(2 * time.Second)
	for errSession.AddInCatalogueError() == "" && time.Now().Before(deadline) {
		time.Sleep(2 * time.Millisecond)
	}
	for i := 0; i < 2; i++ {
		win.BeginFrame()
		drawAddInCatalogueWindow(errSession)
		win.EndFrame(0.1, 0.1, 0.1)
	}
}

// errSource fails every List, to exercise the catalogue header's error path.
type errSource struct{}

func (errSource) List(context.Context, int, int, string) ([]addincat.Entry, error) {
	return nil, context.DeadlineExceeded
}
