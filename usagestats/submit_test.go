// SPDX-License-Identifier: GPL-2.0-only

package usagestats

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func sampleSnapshot() Snapshot {
	return Snapshot{
		MachineUUID: "f1e2d3c4-0000-4000-8000-000000000001",
		OS:          "linux", Arch: "amd64", RAMBytes: 16 << 30, CPU: "AMD Ryzen 7",
		CPUCores: 16, StorageBytes: 512 << 30, GPU: "llvmpipe", VulkanVersion: "1.3.255",
		AppVersion: "0.87.0", AddIns: []AddIn{{ID: "com.oblikovati.cam", Version: "0.6.0"}},
	}
}

// TestSubmitSendsSnapshotAsJSON checks the delegation: Submit marshals the snapshot and POSTs
// it with the CRC token crcpost computes. (crcpost_test covers the transport edge cases.)
func TestSubmitSendsSnapshotAsJSON(t *testing.T) {
	var gotBody []byte
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	if err := NewSubmitter(srv.URL, srv.Client()).Submit(context.Background(), sampleSnapshot()); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	var got Snapshot
	if err := json.Unmarshal(gotBody, &got); err != nil {
		t.Fatalf("server received non-snapshot JSON: %v", err)
	}
	if got.MachineUUID != "f1e2d3c4-0000-4000-8000-000000000001" || got.AppVersion != "0.87.0" {
		t.Errorf("server received wrong snapshot: %+v", got)
	}
	if gotAuth != Token(gotBody) {
		t.Errorf("authorization %q is not the CRC of the body", gotAuth)
	}
}

func TestNewSubmitterDefaultsEndpoint(t *testing.T) {
	sub := NewSubmitter("", http.DefaultClient)
	if sub.endpoint != DefaultEndpoint {
		t.Errorf("endpoint = %q, want default %q", sub.endpoint, DefaultEndpoint)
	}
}

func TestTokenDelegatesToCRC(t *testing.T) {
	if len(Token([]byte("abc"))) != 8 {
		t.Errorf("Token did not return an 8-char CRC hex")
	}
}
