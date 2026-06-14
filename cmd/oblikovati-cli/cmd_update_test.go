// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"strings"
	"testing"

	"oblikovati.org/update"
)

func TestUpdateMessage(t *testing.T) {
	cases := []struct {
		name string
		res  update.Result
		want []string // substrings that must appear
	}{
		{
			name: "available",
			res: update.Result{
				Channel: update.Stable, Current: "0.0.1",
				UpdateAvailable: true,
				Latest:          update.Release{Version: "0.0.2", HTMLURL: "https://gh/r"},
			},
			want: []string{"new stable release", "0.0.2", "https://gh/r"},
		},
		{
			name: "up to date",
			res:  update.Result{Channel: update.Stable, Current: "0.0.2"},
			want: []string{"latest stable release", "0.0.2"},
		},
		{
			name: "offline",
			res:  update.Result{Channel: update.Stable, Skipped: true, SkipReason: "offline"},
			want: []string{"skipped", "no internet connection"},
		},
		{
			name: "dev build",
			res:  update.Result{Channel: update.Dev, Skipped: true, SkipReason: "development build"},
			want: []string{"skipped", "development build"},
		},
		{
			name: "no release",
			res:  update.Result{Channel: update.Nightly, Skipped: true, SkipReason: "no published release"},
			want: []string{"skipped", "nightly channel"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := updateMessage(c.res)
			for _, w := range c.want {
				if !strings.Contains(got, w) {
					t.Errorf("updateMessage(%s) = %q, missing %q", c.name, got, w)
				}
			}
		})
	}
}
