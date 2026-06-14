// SPDX-License-Identifier: GPL-2.0-only

package addinhost

import (
	"strings"
	"testing"

	"oblikovati.org/api"
)

// fixtureAPIMajor is the api major hardcoded in testdata/*/main.go's ObkAddInApiMajor
// (they cannot import oblikovati.org/api — testdata is excluded from the module).
const fixtureAPIMajor = 0

// TestFixturesTrackAPIMajor fails the day the API majors, pointing the maintainer at
// the hardcoded fixture exports so the loader integration tests don't silently start
// exercising the incompatible path instead of the happy path.
func TestFixturesTrackAPIMajor(t *testing.T) {
	if api.Major() != fixtureAPIMajor {
		t.Fatalf("api.Major() = %d but testdata fixtures hardcode %d; update ObkAddInApiMajor in "+
			"testdata/echoaddin & testdata/uiaddin and this constant", api.Major(), fixtureAPIMajor)
	}
}

// TestCheckCompatibility covers the load-time gate: a present, matching-major add-in
// whose minor is not newer than the host loads; a different major, a newer minor, or
// a missing version export is refused with a message naming both versions.
func TestCheckCompatibility(t *testing.T) {
	cases := []struct {
		name                   string
		addinMajor, addinMinor int
		present                bool
		hostMajor, hostMinor   int
		wantErr                string // substring; "" means compatible
	}{
		{"same version loads", 1, 4, true, 1, 4, ""},
		{"older minor loads", 1, 2, true, 1, 4, ""},
		{"zero version loads", 0, 0, true, 0, 1, ""},
		{"different major refused", 1, 0, true, 2, 0, "API major 1, host is API major 2"},
		{"newer minor refused", 1, 5, true, 1, 4, "API 1.5, newer than host API 1.4"},
		{"missing export refused", 0, 0, false, 3, 2, "does not report its API version"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkCompatibility(tc.addinMajor, tc.addinMinor, tc.present, tc.hostMajor, tc.hostMinor)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("checkCompatibility(%d.%d,%v,%d.%d) = %v, want nil",
						tc.addinMajor, tc.addinMinor, tc.present, tc.hostMajor, tc.hostMinor, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("checkCompatibility(%d.%d,%v,%d.%d) = %v, want error containing %q",
					tc.addinMajor, tc.addinMinor, tc.present, tc.hostMajor, tc.hostMinor, err, tc.wantErr)
			}
		})
	}
}
