// SPDX-License-Identifier: GPL-2.0-only

package addincat

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// makeZip builds an in-memory bundle zip from name→content pairs.
func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func sha(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// bundleServer serves the given bytes at /bundle.zip and returns a client + the entry/version
// wired to download it for the current platform.
func bundleServer(t *testing.T, zipBytes []byte) (*Client, Entry, Version, func()) {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(zipBytes)
	}))
	v := Version{Version: "0.6.0", APIMajor: 0, APIMinor: 85, Bundles: map[string]Bundle{
		Platform(): {URL: ts.URL + "/bundle.zip", SHA256: sha(zipBytes), Size: int64(len(zipBytes))},
	}}
	e := Entry{Name: "com.oblikovati.cam", DisplayName: "Oblikovati CAM", Versions: []Version{v}}
	return NewClient(ts.URL, ts.Client()), e, v, ts.Close
}

func newInstaller(t *testing.T, c *Client) *Installer {
	return NewInstaller(t.TempDir(), c)
}

func TestInstallExtractsAndListsInstalled(t *testing.T) {
	zb := makeZip(t, map[string]string{
		"cam/manifest.json": `{"id":"com.oblikovati.cam","version":"0.6.0"}`,
		"cam/libcam.so":     "BINARY",
	})
	c, e, v, closeFn := bundleServer(t, zb)
	defer closeFn()
	in := newInstaller(t, c)

	if err := in.Install(context.Background(), e, v); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, err := os.Stat(filepath.Join(in.dir, "com.oblikovati.cam", "cam", "libcam.so")); err != nil {
		t.Errorf("binary not extracted: %v", err)
	}
	installed, err := in.Installed()
	if err != nil {
		t.Fatalf("Installed: %v", err)
	}
	if len(installed) != 1 || installed[0].Name != "com.oblikovati.cam" || installed[0].Version != "0.6.0" {
		t.Errorf("Installed = %+v", installed)
	}
}

func TestInstallRejectsChecksumMismatch(t *testing.T) {
	c, e, v, closeFn := bundleServer(t, makeZip(t, map[string]string{"manifest.json": `{"id":"x","version":"1"}`}))
	defer closeFn()
	bad := v
	bad.Bundles = map[string]Bundle{Platform(): {URL: v.Bundles[Platform()].URL, SHA256: "deadbeef"}}
	if err := newInstaller(t, c).Install(context.Background(), e, bad); err == nil {
		t.Error("expected a checksum mismatch error")
	}
}

func TestInstallNoBundleForPlatform(t *testing.T) {
	c, e, _, closeFn := bundleServer(t, makeZip(t, map[string]string{"manifest.json": `{"id":"x","version":"1"}`}))
	defer closeFn()
	v := Version{Version: "0.6.0", Bundles: map[string]Bundle{"some-other-os": {URL: "u"}}}
	if err := newInstaller(t, c).Install(context.Background(), e, v); err == nil {
		t.Error("expected an error when no bundle matches the platform")
	}
}

func TestUninstall(t *testing.T) {
	zb := makeZip(t, map[string]string{"cam/manifest.json": `{"id":"com.oblikovati.cam","version":"0.6.0"}`})
	c, e, v, closeFn := bundleServer(t, zb)
	defer closeFn()
	in := newInstaller(t, c)
	if err := in.Install(context.Background(), e, v); err != nil {
		t.Fatal(err)
	}
	if err := in.Uninstall("com.oblikovati.cam"); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	installed, _ := in.Installed()
	if len(installed) != 0 {
		t.Errorf("still installed after uninstall: %+v", installed)
	}
}

func TestStatusClassifies(t *testing.T) {
	zb := makeZip(t, map[string]string{"cam/manifest.json": `{"id":"com.oblikovati.cam","version":"0.6.0"}`})
	c, e, v, closeFn := bundleServer(t, zb)
	defer closeFn()
	in := newInstaller(t, c)

	catalogue := []Entry{
		e, // CAM, will be installed
		{Name: "com.other", DisplayName: "Other", Versions: []Version{{Version: "1.0.0", APIMajor: 0, APIMinor: 85}}},
	}

	// Before install: both Available.
	st, err := in.Status(catalogue, 0, 85)
	if err != nil {
		t.Fatal(err)
	}
	if stateOf(st, "com.oblikovati.cam") != StateAvailable {
		t.Error("CAM should be Available before install")
	}

	// Install 0.6.0, then publish-claim a newer 0.7.0 in the catalogue → UpdateAvailable.
	if err := in.Install(context.Background(), e, v); err != nil {
		t.Fatal(err)
	}
	withUpdate := []Entry{{
		Name: "com.oblikovati.cam", Versions: []Version{{Version: "0.7.0", APIMajor: 0, APIMinor: 85}},
	}}
	st, _ = in.Status(withUpdate, 0, 85)
	if stateOf(st, "com.oblikovati.cam") != StateUpdateAvailable {
		t.Error("CAM should be UpdateAvailable when catalogue has a newer version")
	}

	// Same version in catalogue → Installed (up to date).
	sameVer := []Entry{{Name: "com.oblikovati.cam", Versions: []Version{{Version: "0.6.0", APIMajor: 0, APIMinor: 85}}}}
	st, _ = in.Status(sameVer, 0, 85)
	if stateOf(st, "com.oblikovati.cam") != StateInstalled {
		t.Error("CAM should be Installed when up to date")
	}
}

func TestStatusIncludesInstalledNotInCatalogue(t *testing.T) {
	zb := makeZip(t, map[string]string{"cam/manifest.json": `{"id":"com.local","version":"0.1.0"}`})
	c, _, _, closeFn := bundleServer(t, zb)
	defer closeFn()
	in := newInstaller(t, c)
	if err := in.Install(context.Background(), Entry{Name: "com.local"},
		Version{Version: "0.1.0", Bundles: map[string]Bundle{Platform(): {URL: c.baseURL + "/bundle.zip", SHA256: sha(zb)}}}); err != nil {
		t.Fatal(err)
	}
	st, _ := in.Status(nil, 0, 85) // empty catalogue
	if stateOf(st, "com.local") != StateInstalled {
		t.Error("an installed add-in absent from the catalogue should still appear as Installed")
	}
}

func TestExtractRejectsZipSlip(t *testing.T) {
	zb := makeZip(t, map[string]string{"../escape.txt": "x"})
	if err := extractZip(zb, t.TempDir()); err == nil {
		t.Error("zip-slip entry should be rejected")
	}
}

func TestUnsafeNameRejected(t *testing.T) {
	in := NewInstaller(t.TempDir(), nil)
	if err := in.Uninstall("../escape"); err == nil {
		t.Error("Uninstall accepted an unsafe name")
	}
}

func stateOf(st []AddInStatus, name string) State {
	for _, s := range st {
		if s.Entry.Name == name {
			return s.State
		}
	}
	return State(-1)
}
