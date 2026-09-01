// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"reflect"
	"testing"
)

func TestSelectionRefKind(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"face/abc": "face", "edge/xy": "edge", "vertex/z": "vertex", "garbage": "",
	}
	for ref, want := range cases {
		if got := selectionRefKind(ref); got != want {
			t.Fatalf("selectionRefKind(%q) = %q, want %q", ref, got, want)
		}
	}
}

func TestFilterRefsByAccepts(t *testing.T) {
	t.Parallel()
	refs := []string{"face/a", "edge/b", "vertex/c", "face/d"}
	got := filterRefsByAccepts(refs, []string{"face"})
	if !reflect.DeepEqual(got, []string{"face/a", "face/d"}) {
		t.Fatalf("filter = %v, want [face/a face/d]", got)
	}
	if all := filterRefsByAccepts(refs, nil); !reflect.DeepEqual(all, refs) {
		t.Fatalf("empty accepts should keep all, got %v", all)
	}
}
