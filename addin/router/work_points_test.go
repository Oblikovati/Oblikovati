// SPDX-License-Identifier: GPL-2.0-only

package router

import "testing"

func TestWorkPointsCreateRejectsBadPosition(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "workPoints.create", []byte(`{"at":[1,2]}`)); err == nil {
		t.Error("a 2-component position must error")
	}
}
