// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// TestApplicationApiVersionReportsHostContract proves the runtime query returns the
// host's own api.Version/Major — the same source the load-time handshake gates on, so
// an add-in can never see a version that disagrees with the compatibility check.
func TestApplicationApiVersionReportsHostContract(t *testing.T) {
	r := New(opregistry.Default())
	s := app.NewSession()

	var got wire.ApplicationApiVersionResult
	call(t, r, s, wire.MethodApplicationApiVersion, "{}", &got)

	if got.Version != api.Version {
		t.Errorf("version = %q, want host api.Version %q", got.Version, api.Version)
	}
	if got.Major != api.Major() {
		t.Errorf("major = %d, want host api.Major() %d", got.Major, api.Major())
	}
}
