// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"reflect"
	"testing"
)

func TestComponentGroupsPartitionsAndOrders(t *testing.T) {
	got := ComponentGroups([]int{4, 7, 9, 2}, func(join func(a, b int)) {
		join(7, 2)
		join(9, 9)
		join(11, 4) // unknown id: ignored
	})
	want := [][]int{{4}, {7, 2}, {9}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ComponentGroups = %v, want %v", got, want)
	}
}
