// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// registerApplicationHandlers wires the host-application info methods an add-in can
// query at runtime (the ThisApplication version surface).
func (r *Router) registerApplicationHandlers() {
	r.readOnly(wire.MethodApplicationApiVersion, applicationApiVersion)
}

// applicationApiVersion reports the semantic version of the api contract this host
// implements (wire.MethodApplicationApiVersion). It is sourced from api.Version, the
// same constant the load-time ObkAddInApiMajor handshake gates on, so the runtime
// answer can never disagree with the compatibility check.
func applicationApiVersion(_ *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(wire.ApplicationApiVersionResult{
		Version: api.Version,
		Major:   api.Major(),
		Minor:   api.Minor(),
	})
}
