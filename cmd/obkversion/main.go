// SPDX-License-Identifier: GPL-2.0-only

// Command obkversion prints the application build version under the scheme
// {MANUAL_MAJOR}.{API_VERSION}.{MINOR}.{PATCH} (see package release / RELEASING.md).
// The release and nightly workflows call it to stamp the build:
//
//	go run ./cmd/obkversion stable    # -> 0.000200.1.0
//	go run ./cmd/obkversion nightly   # -> 0.000200.1.0-nightly.20260615T030000
//
// MINOR.PATCH come from git tags + conventional-commit scope and reset to 0.0 when
// MANUAL_MAJOR or API_VERSION change, so the tool needs full history and tags.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"oblikovati.org/release"
)

func main() {
	channel := "stable"
	if len(os.Args) > 1 {
		channel = os.Args[1]
	}
	v, err := compute(channel, ".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "obkversion:", err)
		os.Exit(1)
	}
	fmt.Println(v)
}

// gitRepo answers the two history questions the version needs. The seam keeps assemble
// unit-testable with a fake; main wires the real git in (CLAUDE.md: inject I/O).
type gitRepo interface {
	newestTag(pattern string) string // highest-versioned tag matching pattern, "" if none
	commitsSince(ref string) []string
}

// compute reads the externally-set fields from root and assembles the version against
// the real git history.
func compute(channel, root string) (string, error) {
	major, apiField, err := fixedFields(root)
	if err != nil {
		return "", err
	}
	return assemble(channel, major, apiField, realGit{}, time.Now())
}

// assemble builds the version string from the two fixed fields plus the tag-derived,
// commit-scoped MINOR.PATCH. It is the testable core (repo + now are injected).
func assemble(channel string, major int, apiField string, repo gitRepo, now time.Time) (string, error) {
	prefix := fmt.Sprintf("v%d.%s.", major, apiField)
	baseMinor, basePatch, sinceRef := tagBase(repo, prefix)
	scope := release.Classify(repo.commitsSince(sinceRef))
	minor, patch := release.NextMinorPatch(baseMinor, basePatch, scope)
	v := release.Assemble(major, apiField, minor, patch)

	switch channel {
	case "stable":
		return v, nil
	case "nightly":
		return v + "-nightly." + now.UTC().Format("20060102T150405"), nil
	default:
		return "", fmt.Errorf("unknown channel %q (want stable|nightly)", channel)
	}
}

// fixedFields reads the two externally-set fields: MANUAL_MAJOR (version.yaml) and the
// padded API_VERSION (the api pin in go.mod).
func fixedFields(root string) (int, string, error) {
	yamlBytes, err := os.ReadFile(root + "/version.yaml")
	if err != nil {
		return 0, "", fmt.Errorf("read version.yaml: %w", err)
	}
	major, err := release.ManualMajor(yamlBytes)
	if err != nil {
		return 0, "", err
	}
	goMod, err := os.ReadFile(root + "/go.mod")
	if err != nil {
		return 0, "", fmt.Errorf("read go.mod: %w", err)
	}
	apiVersion, err := release.APIVersionFromGoMod(goMod)
	if err != nil {
		return 0, "", err
	}
	apiField, err := release.APIField(apiVersion)
	if err != nil {
		return 0, "", err
	}
	return major, apiField, nil
}

// tagBase finds the SemVer base for MINOR.PATCH: the newest stable tag on the current
// {major}.{apiField} line. If that line has no tag yet (MAJOR or API just changed),
// MINOR.PATCH reset to 0.0 and the bump is measured since the newest release of any
// line (or all history if the repo has none).
func tagBase(repo gitRepo, prefix string) (minor, patch int, sinceRef string) {
	if tag := repo.newestTag(prefix + "*"); tag != "" {
		if mi, pa, ok := release.ParseVersionTag(tag, prefix); ok {
			return mi, pa, tag
		}
	}
	return 0, 0, repo.newestTag("v*.*.*.*") // a different line — reset, measure since the last release
}

// realGit answers the queries by shelling out to git in the working directory.
type realGit struct{}

func (realGit) newestTag(pattern string) string {
	out, err := runGit("tag", "--list", pattern, "--sort=-v:refname")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(out, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			return t
		}
	}
	return ""
}

// commitsSince returns the non-merge commit messages reachable from HEAD after ref
// (whole history when ref is empty), NUL-separated as `git log --format=%B%x00` emits.
func (realGit) commitsSince(ref string) []string {
	args := []string{"log", "--no-merges", "--format=%B%x00"}
	if ref != "" {
		args = append(args, ref+"..HEAD")
	}
	out, err := runGit(args...)
	if err != nil {
		return nil
	}
	var msgs []string
	for _, m := range strings.Split(out, "\x00") {
		if strings.TrimSpace(m) != "" {
			msgs = append(msgs, m)
		}
	}
	return msgs
}

func runGit(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	return string(out), err
}
