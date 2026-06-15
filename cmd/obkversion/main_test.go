// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"strings"
	"testing"
	"time"
)

// fakeRepo is a named test double for gitRepo: a fixed tag list (newest first) and a
// canned set of commit messages, so assemble is tested without touching real git.
type fakeRepo struct {
	tags    []string            // any order; newestTag returns the first that matches
	commits map[string][]string // sinceRef -> messages ("" key = whole history)
}

func (f fakeRepo) newestTag(pattern string) string {
	want := strings.TrimSuffix(pattern, "*")
	for _, t := range f.tags {
		if strings.HasPrefix(t, want) || pattern == "v*.*.*.*" {
			return t
		}
	}
	return ""
}

func (f fakeRepo) commitsSince(ref string) []string { return f.commits[ref] }

func TestAssemble(t *testing.T) {
	noon := time.Date(2026, 6, 15, 3, 0, 0, 0, time.UTC)
	cases := []struct {
		name    string
		repo    fakeRepo
		channel string
		want    string
	}{
		{
			name:    "first release on a new api line (no matching tag, feature)",
			repo:    fakeRepo{commits: map[string][]string{"": {"feat: thing"}}},
			channel: "stable",
			want:    "0.000200.1.0",
		},
		{
			name: "patch bump from the line's latest tag",
			repo: fakeRepo{
				tags:    []string{"v0.000200.1.0"},
				commits: map[string][]string{"v0.000200.1.0": {"fix: bug"}},
			},
			channel: "stable",
			want:    "0.000200.1.1",
		},
		{
			name: "feature bump resets patch",
			repo: fakeRepo{
				tags:    []string{"v0.000200.1.4"},
				commits: map[string][]string{"v0.000200.1.4": {"feat: x", "fix: y"}},
			},
			channel: "stable",
			want:    "0.000200.2.0",
		},
		{
			name: "api change resets MINOR.PATCH, measured since the last release",
			repo: fakeRepo{
				tags:    []string{"v0.000100.5.2"}, // a DIFFERENT api line
				commits: map[string][]string{"v0.000100.5.2": {"fix: small"}},
			},
			channel: "stable",
			want:    "0.000200.0.1", // reset to 0.0, then a fix
		},
		{
			name:    "nightly appends the timestamp",
			repo:    fakeRepo{commits: map[string][]string{"": {"feat: thing"}}},
			channel: "nightly",
			want:    "0.000200.1.0-nightly.20260615T030000",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := assemble(c.channel, 0, "000200", c.repo, noon)
			if err != nil || got != c.want {
				t.Fatalf("assemble = %q, %v; want %q", got, err, c.want)
			}
		})
	}
}

func TestAssembleRejectsUnknownChannel(t *testing.T) {
	if _, err := assemble("beta", 0, "000200", fakeRepo{}, time.Now()); err == nil {
		t.Fatal("assemble with an unknown channel should error")
	}
}
