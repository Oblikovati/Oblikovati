// SPDX-License-Identifier: GPL-2.0-only

// Package release computes the application's build version under the scheme
//
//		{MANUAL_MAJOR}.{API_VERSION}.{MINOR}.{PATCH}
//
//	  - MANUAL_MAJOR is hand-set in version.yaml (a deliberate generational bump);
//	  - API_VERSION is the referenced oblikovati.org/api release, each semver
//	    component zero-padded to two digits and concatenated (v0.2.0 -> "000200");
//	  - MINOR and PATCH auto-number from conventional-commit scope (SemVer rules),
//	    read from git tags and RESET to 0.0 whenever MANUAL_MAJOR or API_VERSION change.
//
// The functions here are pure (they take already-read bytes / strings); cmd/obkversion
// supplies the file and git I/O. See RELEASING.md and architecture/decisions/ADR-0017.
package release

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ManualMajor parses the MANUAL_MAJOR from version.yaml (`major: N`).
func ManualMajor(versionYAML []byte) (int, error) {
	var doc struct {
		Major *int `yaml:"major"`
	}
	if err := yaml.Unmarshal(versionYAML, &doc); err != nil {
		return 0, fmt.Errorf("version.yaml is not valid YAML: %w", err)
	}
	if doc.Major == nil || *doc.Major < 0 {
		return 0, fmt.Errorf("version.yaml must set a non-negative `major:` (got %v)", doc.Major)
	}
	return *doc.Major, nil
}

var apiRequireRe = regexp.MustCompile(`oblikovati\.org/api\s+v([0-9]+\.[0-9]+\.[0-9]+)`)

// APIVersionFromGoMod extracts the pinned oblikovati.org/api version from go.mod,
// e.g. "0.2.0". The pin is kept current by scripts/sync-api-version.sh.
func APIVersionFromGoMod(goMod []byte) (string, error) {
	m := apiRequireRe.FindSubmatch(goMod)
	if m == nil {
		return "", fmt.Errorf("go.mod has no `require oblikovati.org/api vX.Y.Z` line")
	}
	return string(m[1]), nil
}

// APIField renders an api semver ("v0.2.0" or "0.2.0") as the API_VERSION field: each
// component zero-padded to at least two digits and concatenated. v0.2.0 -> "000200",
// v0.12.3 -> "001203".
func APIField(apiVersion string) (string, error) {
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(apiVersion), "v"), ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("api version %q is not MAJOR.MINOR.PATCH", apiVersion)
	}
	var b strings.Builder
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return "", fmt.Errorf("api version %q has a non-numeric component %q", apiVersion, p)
		}
		fmt.Fprintf(&b, "%02d", n)
	}
	return b.String(), nil
}

// Assemble joins the four fields into the full version string, e.g. "0.000200.1.0".
func Assemble(major int, apiField string, minor, patch int) string {
	return fmt.Sprintf("%d.%s.%d.%d", major, apiField, minor, patch)
}
