//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"oblikovati.org/build"
	"oblikovati.org/usagestats"
)

func TestAssembleSnapshotFillsPlatformAndRenderer(t *testing.T) {
	t.Setenv("OBK_USER_GLOBALS_FILE", filepath.Join(t.TempDir(), "globals"))
	t.Setenv("OBK_USER_ADDINS_DIR", t.TempDir()) // no add-ins installed

	snap := assembleSnapshot("llvmpipe", "1.3.255")
	if snap.OS != runtime.GOOS || snap.Arch != runtime.GOARCH {
		t.Errorf("OS/Arch = %s/%s, want %s/%s", snap.OS, snap.Arch, runtime.GOOS, runtime.GOARCH)
	}
	if snap.GPU != "llvmpipe" || snap.VulkanVersion != "1.3.255" {
		t.Errorf("renderer fields not carried: GPU=%q Vulkan=%q", snap.GPU, snap.VulkanVersion)
	}
	if snap.AppVersion != build.Version {
		t.Errorf("AppVersion = %q, want build.Version %q", snap.AppVersion, build.Version)
	}
	if snap.MachineUUID == "" {
		t.Error("MachineUUID should be generated")
	}
	if snap.CPUCores < 1 {
		t.Errorf("CPUCores = %d, want >= 1", snap.CPUCores)
	}
}

func TestInstalledAddInsForTelemetryReadsUserDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OBK_USER_ADDINS_DIR", dir)
	// Two installed add-ins, each a subdir with a manifest.json.
	writeManifest(t, filepath.Join(dir, "cam"), `{"id":"com.oblikovati.cam","version":"0.6.0"}`)
	writeManifest(t, filepath.Join(dir, "mcp"), `{"id":"com.oblikovati.mcp-bridge","version":"0.2.0"}`)

	got := installedAddInsForTelemetry()
	if len(got) != 2 {
		t.Fatalf("found %d add-ins, want 2: %+v", len(got), got)
	}
	ids := map[string]string{}
	for _, a := range got {
		ids[a.ID] = a.Version
	}
	if ids["com.oblikovati.cam"] != "0.6.0" || ids["com.oblikovati.mcp-bridge"] != "0.2.0" {
		t.Errorf("add-in id/version map wrong: %v", ids)
	}
}

func TestInstalledAddInsForTelemetryEmptyDir(t *testing.T) {
	t.Setenv("OBK_USER_ADDINS_DIR", t.TempDir())
	if got := installedAddInsForTelemetry(); len(got) != 0 {
		t.Errorf("empty dir yielded %d add-ins, want 0", len(got))
	}
}

func TestSubmitSnapshotPostsToConfiguredEndpoint(t *testing.T) {
	var gotAuth string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()
	t.Setenv("OBLIKOVATI_STATS_ENDPOINT", srv.URL)

	snap := usagestats.Snapshot{MachineUUID: "m", OS: "linux", Arch: "amd64", AppVersion: "0.1.0"}
	if err := submitSnapshot(snap); err != nil {
		t.Fatalf("submitSnapshot: %v", err)
	}
	if len(gotBody) == 0 {
		t.Fatal("server received no body")
	}
	if gotAuth != usagestats.Token(gotBody) {
		t.Errorf("authorization %q is not the CRC of the body", gotAuth)
	}
}

func writeManifest(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}
