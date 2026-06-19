// SPDX-License-Identifier: GPL-2.0-only

package router

import "testing"

// TestBrepHandlersRejectMissingHandle covers the "no transient body with handle" branches: every
// handle-addressed brep op rejects an unknown handle (999 was never created).
func TestBrepHandlersRejectMissingHandle(t *testing.T) {
	r, s := seededSession(t)
	const bad = `{"handle":999}`
	cases := map[string]string{
		"brep.transform":   `{"handle":999,"matrix":[1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1]}`,
		"brep.deleteFaces": `{"handle":999,"faces":[0]}`,
		"brep.describe":    bad,
		"brep.delete":      bad,
		"brep.copy":        bad, // exercises brepSource's missing-handle ref path
	}
	for method, args := range cases {
		if err := tryCall(t, r, s, method, args); err == nil {
			t.Errorf("%s with an unknown handle should error", method)
		}
	}
}
