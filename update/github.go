// SPDX-License-Identifier: GPL-2.0-only

package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// httpDoer is the one method of *http.Client this package needs, named so tests can fake
// the transport without a live server (CLAUDE.md: wrap external I/O behind a thin seam).
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// GitHubSource reads releases from the GitHub REST API. It maps a [Channel] to the right
// endpoint — stable to the repo's "latest" release, nightly to the rolling `nightly`
// tag — mirroring how release.yml and nightly.yml publish them.
type GitHubSource struct {
	owner, repo string
	apiBase     string // overridable for tests; defaults to the public API host
	http        httpDoer
}

// DefaultOwner/DefaultRepo are the published Oblikovati repository on GitHub.
const (
	DefaultOwner = "Oblikovati"
	DefaultRepo  = "Oblikovati"
)

// defaultAPIBase is the public GitHub REST API host.
const defaultAPIBase = "https://api.github.com"

// maxReleaseBody caps the JSON we read from GitHub (a release object is a few KiB); it
// bounds memory if the endpoint ever misbehaves.
const maxReleaseBody = 1 << 20

// NewGitHubSource returns a source for owner/repo over doer (typically an *http.Client
// with a short timeout so a slow network never blocks startup).
func NewGitHubSource(owner, repo string, doer httpDoer) *GitHubSource {
	return &GitHubSource{owner: owner, repo: repo, apiBase: defaultAPIBase, http: doer}
}

// Latest fetches the newest release on channel, translating connectivity failures to
// [ErrOffline] and a missing release to [ErrNoRelease] so the checker can skip them.
func (g *GitHubSource) Latest(ctx context.Context, channel Channel) (Release, error) {
	raw, err := g.get(ctx, g.endpoint(channel))
	if err != nil {
		return Release{}, err
	}
	return parseRelease(raw, channel)
}

// endpoint is the releases API URL for channel: the moving `nightly` tag for nightly, the
// "latest" (newest non-prerelease) for stable.
func (g *GitHubSource) endpoint(channel Channel) string {
	if channel == Nightly {
		return fmt.Sprintf("%s/repos/%s/%s/releases/tags/nightly", g.apiBase, g.owner, g.repo)
	}
	return fmt.Sprintf("%s/repos/%s/%s/releases/latest", g.apiBase, g.owner, g.repo)
}

// get performs the request and returns the body, mapping transport errors to ErrOffline
// and a 404 to ErrNoRelease.
func (g *GitHubSource) get(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("update: build request for %q: %w", endpoint, err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := g.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOffline, err) // no network / DNS / timeout
	}
	defer resp.Body.Close()
	return readReleaseBody(resp, endpoint)
}

// readReleaseBody validates the status and returns the capped body.
func readReleaseBody(resp *http.Response, endpoint string) ([]byte, error) {
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNoRelease
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update: GET %q: %s", endpoint, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxReleaseBody))
}

// ghRelease is the subset of GitHub's release object we use.
type ghRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
}

// parseRelease turns a GitHub release JSON into a [Release], deriving the comparable
// version from the right field for the channel.
func parseRelease(raw []byte, channel Channel) (Release, error) {
	var gr ghRelease
	if err := json.Unmarshal(raw, &gr); err != nil {
		return Release{}, fmt.Errorf("update: parse release JSON: %w", err)
	}
	ver := releaseVersion(gr, channel)
	if ver == "" {
		return Release{}, fmt.Errorf("update: release has no version (tag=%q name=%q)", gr.TagName, gr.Name)
	}
	return Release{Version: ver, HTMLURL: gr.HTMLURL, Channel: channel}, nil
}

// releaseVersion reads the comparable version: a stable release carries it in the
// "v<version>" tag, but the nightly tag is the fixed string "nightly", so its version
// lives in the title ("Nightly <version>"), set by nightly.yml.
func releaseVersion(gr ghRelease, channel Channel) string {
	if channel == Nightly {
		return lastField(gr.Name)
	}
	return strings.TrimPrefix(gr.TagName, "v")
}

// lastField returns the final whitespace-separated token of s (the version in a
// "Nightly <version>" title), or "" when s has none.
func lastField(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}
