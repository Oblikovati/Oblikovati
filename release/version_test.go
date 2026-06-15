// SPDX-License-Identifier: GPL-2.0-only

package release

import "testing"

func TestManualMajor(t *testing.T) {
	got, err := ManualMajor([]byte("major: 0\n"))
	if err != nil || got != 0 {
		t.Fatalf("ManualMajor(major: 0) = %d, %v; want 0, nil", got, err)
	}
	if got, err := ManualMajor([]byte("major: 3\n")); err != nil || got != 3 {
		t.Fatalf("ManualMajor(major: 3) = %d, %v; want 3, nil", got, err)
	}
	for _, bad := range []string{"", "name: x\n", "major: -1\n", "major: oops\n"} {
		if _, err := ManualMajor([]byte(bad)); err == nil {
			t.Errorf("ManualMajor(%q) = nil error, want a rejection", bad)
		}
	}
}

func TestAPIVersionFromGoMod(t *testing.T) {
	mod := "module oblikovati.org\n\nrequire (\n\toblikovati.org/api v0.2.0\n)\n"
	if got, err := APIVersionFromGoMod([]byte(mod)); err != nil || got != "0.2.0" {
		t.Fatalf("APIVersionFromGoMod = %q, %v; want 0.2.0", got, err)
	}
	if _, err := APIVersionFromGoMod([]byte("module x\n")); err == nil {
		t.Error("APIVersionFromGoMod without the api require should error")
	}
}

func TestAPIField(t *testing.T) {
	cases := map[string]string{"v0.2.0": "000200", "0.2.0": "000200", "v0.12.3": "001203", "v1.0.0": "010000"}
	for in, want := range cases {
		if got, err := APIField(in); err != nil || got != want {
			t.Errorf("APIField(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"0.2", "v1.2.3.4", "0.x.0", ""} {
		if _, err := APIField(bad); err == nil {
			t.Errorf("APIField(%q) = nil error, want a rejection", bad)
		}
	}
}

func TestAssemble(t *testing.T) {
	if got := Assemble(0, "000200", 1, 0); got != "0.000200.1.0" {
		t.Errorf("Assemble = %q, want 0.000200.1.0", got)
	}
}

func TestParseVersionTag(t *testing.T) {
	const prefix = "v0.000200."
	if mi, pa, ok := ParseVersionTag("v0.000200.3.5", prefix); !ok || mi != 3 || pa != 5 {
		t.Errorf("ParseVersionTag matching = %d,%d,%v; want 3,5,true", mi, pa, ok)
	}
	for _, no := range []string{"v0.000100.3.5", "v0.000200.3", "v0.000200.3.5.1", "v0.000200.x.5", "nightly"} {
		if _, _, ok := ParseVersionTag(no, prefix); ok {
			t.Errorf("ParseVersionTag(%q) ok=true, want false", no)
		}
	}
}
