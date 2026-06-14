// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strings"
	"testing"

	"oblikovati.org/update"
)

func TestUpdateHeadline(t *testing.T) {
	cases := []struct {
		name string
		res  update.Result
		want string
	}{
		{
			name: "available",
			res: update.Result{
				Channel: update.Nightly, UpdateAvailable: true,
				Latest: update.Release{Version: "0.0.20260615030000-nightly"},
			},
			want: "new nightly release is available: 0.0.20260615030000-nightly",
		},
		{
			name: "up to date",
			res:  update.Result{Channel: update.Stable, Current: "0.0.1"},
			want: "up to date (stable 0.0.1)",
		},
		{name: "offline", res: skipResult("offline", update.Stable), want: "no internet connection"},
		{name: "dev", res: skipResult("development build", update.Dev), want: "development build"},
		{name: "no release", res: skipResult("no published release", update.Nightly), want: "nightly channel yet"},
		{name: "failed", res: skipResult("check failed", update.Stable), want: "could not reach the update server"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := updateHeadline(&c.res); !strings.Contains(got, c.want) {
				t.Errorf("updateHeadline(%s) = %q, missing %q", c.name, got, c.want)
			}
		})
	}
}

func skipResult(reason string, ch update.Channel) update.Result {
	return update.Result{Channel: ch, Skipped: true, SkipReason: reason}
}
