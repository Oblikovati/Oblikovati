// SPDX-License-Identifier: GPL-2.0-only

package gopherlua

import (
	"context"
	"testing"
	"time"

	"oblikovati.org/script"
)

// TestCallHostFailureRaisesLuaError covers the error-raising branch of invokeMethod: when the
// host call returns an error, oblikovati.call surfaces it as a Lua error naming the method.
func TestCallHostFailureRaisesLuaError(t *testing.T) {
	f := &fakeCall{err: errBoom}
	res := New().Run(context.Background(),
		`oblikovati.call("documents.create", { type = "part" })`,
		globalsWith(f), script.Limits{Wall: time.Second})
	if res.Err == nil {
		t.Fatal("a failing host call should surface a script error")
	}
}
