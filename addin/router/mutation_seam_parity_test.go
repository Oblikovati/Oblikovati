// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"strings"
	"testing"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/app"
)

// The B1 regression tests (#1612): the wire router and the head UI's Session
// verbs must produce the SAME outcome for the same mutation, because both now
// finish through one seam whose invariant lives on the aggregate. Before the
// fix, parameters.delete refused an in-use parameter while
// Session.DeleteParameter deleted it unconditionally ("dependents go sick").

// deleteInUseOverBothDrivers drives the same in-use delete through one driver
// and reports the outcome; the parameter must survive either way.
func deleteInUseOverBothDrivers(t *testing.T, drive func(r *Router, s *app.Session) error) error {
	t.Helper()
	r, s := seededSession(t)
	call(t, r, s, "parameters.add", `{"name":"half","expression":"width / 2"}`, nil)
	err := drive(r, s)
	holder, findErr := modelaccess.ActiveParameterHolder(s)
	if findErr != nil {
		t.Fatalf("resolving the holder: %v", findErr)
	}
	if _, ok := holder.Parameters().ByName("width"); !ok {
		t.Fatal("width was deleted while in use — the invariant did not hold on this driver")
	}
	return err
}

// TestParameterDeleteInvariantSharedAcrossDrivers is the test that would have
// caught B1: both drivers refuse the in-use delete and name the blocker.
func TestParameterDeleteInvariantSharedAcrossDrivers(t *testing.T) {
	wireErr := deleteInUseOverBothDrivers(t, func(r *Router, s *app.Session) error {
		_, err := r.Handle(s, "parameters.delete", []byte(`{"name":"width"}`))
		return err
	})
	uiErr := deleteInUseOverBothDrivers(t, func(r *Router, s *app.Session) error {
		holder, err := modelaccess.ActiveParameterHolder(s)
		if err != nil {
			return err
		}
		p, _ := holder.Parameters().ByName("width")
		return s.DeleteParameter(p.ID())
	})
	for driver, err := range map[string]error{"wire": wireErr, "ui": uiErr} {
		if err == nil || !strings.Contains(err.Error(), "half") {
			t.Errorf("%s delete(in-use) err = %v, want a refusal naming the blocker \"half\"", driver, err)
		}
	}
}

// TestGroupRenameInvariantSharedAcrossDrivers: the empty-display-name refusal
// comes from the aggregate on both drivers (it used to live only in the wire
// handler while Session.RenameParameterGroup wrote the field unchecked).
func TestGroupRenameInvariantSharedAcrossDrivers(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "parameters.groups.add", `{"internalName":"frame","displayName":"Frame"}`, nil)

	if _, err := r.Handle(s, "parameters.groups.setDisplayName", []byte(`{"internalName":"frame","displayName":""}`)); err == nil {
		t.Error("wire setDisplayName(\"\") must be refused")
	}
	if err := s.RenameParameterGroup("frame", ""); err == nil {
		t.Error("ui RenameParameterGroup(\"\") must be refused")
	}
	holder, err := modelaccess.ActiveParameterHolder(s)
	if err != nil {
		t.Fatalf("resolving the holder: %v", err)
	}
	g, ok := holder.Parameters().GroupByKey("frame")
	if !ok || g.DisplayName() != "Frame" {
		t.Fatalf("group after refused renames = %+v, want display name Frame kept", g)
	}
}
