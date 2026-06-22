// SPDX-License-Identifier: GPL-2.0-only

package app

// Window-open requests for head-only windows reachable from a ribbon button. A command runs in
// the headless core, which cannot draw a window itself, so it raises a one-shot flag here; the
// head consumes the flag each frame and opens the corresponding window. This mirrors the
// file-dialog request pattern (e.g. RequestImportMesh) for the AddIn Catalogue and Preferences,
// which the Get Started ribbon's Manage panel exposes as launch buttons.

// RequestAddInCatalogue flags that the user asked to open the Add-In Catalogue window.
func (s *Session) RequestAddInCatalogue() { s.addInCatalogueRequested = true }

// TakeAddInCatalogueRequest returns and clears the pending Add-In Catalogue request (one-shot).
func (s *Session) TakeAddInCatalogueRequest() bool {
	req := s.addInCatalogueRequested
	s.addInCatalogueRequested = false
	return req
}

// RequestPreferences flags that the user asked to open the Preferences window.
func (s *Session) RequestPreferences() { s.preferencesRequested = true }

// TakePreferencesRequest returns and clears the pending Preferences request (one-shot).
func (s *Session) TakePreferencesRequest() bool {
	req := s.preferencesRequested
	s.preferencesRequested = false
	return req
}
