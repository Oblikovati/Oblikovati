// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app/markingmenu"
)

var errMarkingMenuSave = errors.New("marking menu save failed")

type failingMarkingMenuStore struct{}

func (failingMarkingMenuStore) Load() (markingmenu.Customization, error) {
	return markingmenu.Defaults(), nil
}

func (failingMarkingMenuStore) Save(markingmenu.Customization) error {
	return errMarkingMenuSave
}

func newFailingMarkingMenuSession(t *testing.T) *Session {
	t.Helper()
	s := NewSession()
	if err := s.UseMarkingMenuStore(failingMarkingMenuStore{}); err != nil {
		t.Fatalf("UseMarkingMenuStore: %v", err)
	}
	return s
}

func TestSetMarkingMenuReportsSaveError(t *testing.T) {
	s := newFailingMarkingMenuSession(t)
	err := s.SetMarkingMenu(wire.MarkingMenuView{
		Environment: BaseEnvironment,
		Quadrants:   []wire.MarkingMenuItem{{Quadrant: types.QuadrantNorth, CommandID: "Sketch.Create2D"}},
	})
	if !errors.Is(err, errMarkingMenuSave) {
		t.Fatalf("SetMarkingMenu error = %v, want %v", err, errMarkingMenuSave)
	}
}

func TestMarkingMenuStyleSaveErrorReachesNotice(t *testing.T) {
	s := newFailingMarkingMenuSession(t)
	s.ToggleContextMenuStyle()
	if want := "marking menu: " + errMarkingMenuSave.Error(); s.Notice() != want {
		t.Fatalf("Notice() = %q, want %q", s.Notice(), want)
	}
}
