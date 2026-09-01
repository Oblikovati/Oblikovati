// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestWorkPlaneFixedFrame covers the fixed-frame work-plane constructor (origin + two
// axis vectors), which the other work-plane tests do not exercise.
func TestWorkPlaneFixedFrame(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	var res wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create",
		`{"kind":"fixed-frame","origin":[0,0,5],"xaxis":[1,0,0],"yaxis":[0,1,0]}`, &res)

	// A degenerate axis vector is a clean error.
	if err := tryCall(t, r, s, "workPlanes.create",
		`{"kind":"fixed-frame","origin":[0,0,5],"xaxis":[0,0,0],"yaxis":[0,1,0]}`); err == nil {
		t.Error("a zero x-axis should error")
	}
}
