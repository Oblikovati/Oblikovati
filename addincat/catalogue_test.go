// SPDX-License-Identifier: GPL-2.0-only

package addincat

import (
	"encoding/json"
	"strings"
	"testing"
)

// serviceWireSample is a catalogue record encoded with the EXACT JSON keys the
// addins.oblikovati.org service emits. Decoding it into the host's duplicated Entry pins the
// two copies of the DTO together (the report.Payload precedent): a key rename on either side
// fails this test.
const serviceWireSample = `{
  "name": "com.oblikovati.cam",
  "displayName": "Oblikovati CAM",
  "description": "machining",
  "license": "GPL-2.0-only",
  "iconUrl": "https://addins.oblikovati.org/a/com.oblikovati.cam/icon.svg",
  "sourceRepo": "https://github.com/Oblikovati/Oblikovati.AddIns.CAM",
  "images": ["https://addins.oblikovati.org/a/com.oblikovati.cam/images/shot.png"],
  "versions": [
    {
      "version": "0.6.0",
      "apiMajor": 0,
      "apiMinor": 85,
      "bundles": {
        "linux-amd64": {
          "url": "https://addins.oblikovati.org/a/com.oblikovati.cam/versions/0.6.0/linux-amd64/bundle.zip",
          "sha256": "abc123",
          "size": 794
        }
      },
      "publishedAt": "2026-06-21T13:18:35Z"
    }
  ]
}`

func TestWireFormatRoundTrip(t *testing.T) {
	var e Entry
	if err := json.Unmarshal([]byte(serviceWireSample), &e); err != nil {
		t.Fatalf("decode service sample: %v", err)
	}
	if e.Name != "com.oblikovati.cam" || e.License != "GPL-2.0-only" || e.IconURL == "" || e.SourceRepo == "" {
		t.Errorf("metadata not decoded: %+v", e)
	}
	if len(e.Images) != 1 || len(e.Versions) != 1 {
		t.Fatalf("images/versions not decoded: %+v", e)
	}
	v := e.Versions[0]
	b := v.Bundles["linux-amd64"]
	if v.Version != "0.6.0" || v.APIMajor != 0 || v.APIMinor != 85 || b.SHA256 != "abc123" || b.Size != 794 {
		t.Errorf("version/bundle not decoded: %+v / %+v", v, b)
	}

	// Re-marshalling must keep the service's keys (so a host upload/echo stays compatible).
	out, _ := json.Marshal(e)
	for _, key := range []string{`"displayName"`, `"iconUrl"`, `"sourceRepo"`, `"apiMajor"`, `"apiMinor"`, `"bundles"`, `"sha256"`} {
		if !strings.Contains(string(out), key) {
			t.Errorf("re-marshalled entry is missing key %s", key)
		}
	}
}

func TestLatestFor(t *testing.T) {
	e := Entry{Versions: []Version{
		{Version: "0.5.0", APIMajor: 0, APIMinor: 84},
		{Version: "0.6.0", APIMajor: 0, APIMinor: 85},
		{Version: "0.7.0", APIMajor: 0, APIMinor: 85},
		{Version: "1.0.0", APIMajor: 1, APIMinor: 0},
	}}
	v, ok := e.LatestFor(0, 85)
	if !ok || v.Version != "0.7.0" {
		t.Errorf("LatestFor(0,85) = (%q,%v), want 0.7.0", v.Version, ok)
	}
	if _, ok := e.LatestFor(0, 99); ok {
		t.Error("LatestFor(0,99) should not match")
	}
}

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.6.0", "0.6.1", -1}, {"0.10.0", "0.9.0", 1}, {"1.0.0", "1.0.0", 0}, {"1.2.3-rc1", "1.2.3", 0},
	}
	for _, c := range cases {
		if got := compareSemver(c.a, c.b); got != c.want {
			t.Errorf("compareSemver(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
