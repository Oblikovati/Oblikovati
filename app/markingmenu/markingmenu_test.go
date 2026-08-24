// SPDX-License-Identifier: GPL-2.0-only

package markingmenu

import (
	"path/filepath"
	"reflect"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

func TestToCustomizationSortsEnvironments(t *testing.T) {
	menus := map[types.Environment]wire.MarkingMenuView{
		types.Environment(3): {Environment: types.Environment(3)},
		types.Environment(1): {Environment: types.Environment(1)},
		types.Environment(2): {Environment: types.Environment(2)},
	}
	got := ToCustomization(menus, false)
	if want := []int{1, 2, 3}; !reflect.DeepEqual(environmentIDs(got), want) {
		t.Fatalf("environment order = %v, want %v", environmentIDs(got), want)
	}
}

func environmentIDs(c Customization) []int {
	ids := make([]int, len(c.Menus))
	for i, menu := range c.Menus {
		ids[i] = menu.Environment
	}
	return ids
}

func TestFileStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "markingmenu.yaml")
	store := NewFileStore(path)
	want := Customization{
		Classic: true,
		Menus: []MenuEntry{{
			Environment: int(types.BaseEnvironment),
			Quadrants:   []SlotEntry{{Quadrant: int(types.QuadrantNorth), Command: "Sketch.Create2D"}},
			Overflow:    []string{"Repeat"},
		}},
	}
	if err := store.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}
}
