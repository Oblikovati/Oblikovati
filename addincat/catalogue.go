// SPDX-License-Identifier: GPL-2.0-only

// Package addincat is the host side of the add-in catalogue (#1164): a thin client for the
// addins.oblikovati.org service plus an installer that downloads a bundle into the per-user
// add-ins directory. The application's Add-In Catalogue window drives it; the loader scans
// the same directory at startup.
//
// The Entry/Version/Bundle types here DUPLICATE the service's catalogue DTO (the service is
// a separate, stdlib-only module). The wire format is pinned by a round-trip test so the two
// copies cannot drift — the same precedent as report.Payload.
package addincat

// Bundle is one downloadable per-platform artifact. SHA256 lets the installer verify the
// download before extracting it.
type Bundle struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// Version is one published release: the Oblikovati API major+minor it targets and a bundle
// per platform ("linux-amd64", "darwin-arm64", …).
type Version struct {
	Version     string            `json:"version"`
	APIMajor    int               `json:"apiMajor"`
	APIMinor    int               `json:"apiMinor"`
	Bundles     map[string]Bundle `json:"bundles"`
	PublishedAt string            `json:"publishedAt"`
}

// Entry is one add-in's catalogue record. Name is the unique add-in id (also the install
// directory name); DisplayName is the human label.
type Entry struct {
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	Description string    `json:"description"`
	License     string    `json:"license"`
	IconURL     string    `json:"iconUrl"`
	SourceRepo  string    `json:"sourceRepo"`
	Images      []string  `json:"images,omitempty"`
	Versions    []Version `json:"versions"`
}

// LatestFor returns the newest version (by semantic version) whose API compatibility equals
// major+minor — what a host on that API version installs — and true when one exists.
func (e Entry) LatestFor(major, minor int) (Version, bool) {
	var best Version
	found := false
	for _, v := range e.Versions {
		if v.APIMajor != major || v.APIMinor != minor {
			continue
		}
		if !found || compareSemver(v.Version, best.Version) > 0 {
			best, found = v, true
		}
	}
	return best, found
}
