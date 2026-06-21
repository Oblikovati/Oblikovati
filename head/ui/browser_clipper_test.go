//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// TestLeafRunLengths pins how drawChildren partitions a child list into the contiguous leaf runs it
// virtualizes: branches break runs, leaves between/around them form runs (M34-F3). This is the
// flat-assembly case — a long run of occurrence leaves beside the Origin/Parameters branches.
func TestLeafRunLengths(t *testing.T) {
	branch := app.BrowserNode{Label: "p", Children: []app.BrowserNode{{Label: "c"}}}
	leaf := app.BrowserNode{Label: "leaf"}
	cases := []struct {
		name string
		in   []app.BrowserNode
		want []int
	}{
		{"all leaves", []app.BrowserNode{leaf, leaf, leaf}, []int{3}},
		{"leading branches then run", []app.BrowserNode{branch, branch, leaf, leaf, leaf}, []int{3}},
		{"runs split by a branch", []app.BrowserNode{leaf, leaf, branch, leaf}, []int{2, 1}},
		{"only branches", []app.BrowserNode{branch, branch}, nil},
		{"empty", nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := leafRunLengths(c.in)
			if len(got) != len(c.want) {
				t.Fatalf("runs = %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("run %d = %d, want %d", i, got[i], c.want[i])
				}
			}
		})
	}
}

// TestInWindowBrowserClipsWideLeafList drives the real clipper path in an open Dear ImGui window: a
// wide all-leaf list is drawn through drawChildren over several frames. A mismatched clipper
// Begin/Step/End or a bad per-row id would trip ImGui's assertions, and the test pins the id
// invariant — browserNodeSeq must advance by exactly one per leaf (no descendants), regardless of
// how many rows the clipper actually submitted, so sibling nodes after the list keep their ids.
func TestInWindowBrowserClipsWideLeafList(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	s := app.NewSession()

	const n = 600 // well over browserClipThreshold
	leaves := make([]app.BrowserNode, n)
	for i := range leaves {
		leaves[i] = app.BrowserNode{Label: fmt.Sprintf("body %d", i), Kind: "body"}
	}

	for f := 0; f < 3; f++ {
		win.BeginFrame()
		native.Begin("clip-test")
		browserNodeSeq = 7
		drawChildren(s, leaves)
		got := browserNodeSeq
		native.End()
		win.EndFrame(0, 0, 0)
		if got != 7+n {
			t.Fatalf("browserNodeSeq = %d after a clipped %d-leaf list, want %d (one id per leaf)", got, n, 7+n)
		}
	}
}

// TestInWindowBrowserSmallListUnclipped checks the sub-threshold path still draws and advances the
// id counter by one per leaf (the recursive walk), so the two paths agree on id accounting.
func TestInWindowBrowserSmallListUnclipped(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	s := app.NewSession()

	const n = 5 // under browserClipThreshold → recursive path
	leaves := make([]app.BrowserNode, n)
	for i := range leaves {
		leaves[i] = app.BrowserNode{Label: fmt.Sprintf("leaf %d", i)}
	}
	win.BeginFrame()
	native.Begin("small-test")
	browserNodeSeq = 0
	drawChildren(s, leaves)
	got := browserNodeSeq
	native.End()
	win.EndFrame(0, 0, 0)
	if got != n {
		t.Fatalf("browserNodeSeq = %d after a %d-leaf recursive list, want %d", got, n, n)
	}
}
