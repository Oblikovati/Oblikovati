// SPDX-License-Identifier: GPL-2.0-only

package usagestats

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"strings"
	"testing"
)

// fakeDoer captures the request it is handed and replies with a canned response (or a
// transport error), so Submit can be exercised without a live server.
type fakeDoer struct {
	gotReq  *http.Request
	gotBody []byte
	status  int
	body    string
	err     error
}

func (f *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.gotReq = req
	if req.Body != nil {
		f.gotBody, _ = io.ReadAll(req.Body)
	}
	return &http.Response{
		StatusCode: f.status,
		Status:     fmt.Sprintf("%d %s", f.status, http.StatusText(f.status)),
		Body:       io.NopCloser(strings.NewReader(f.body)),
		Header:     make(http.Header),
	}, nil
}

func sampleSnapshot() Snapshot {
	return Snapshot{
		MachineUUID: "f1e2d3c4-0000-4000-8000-000000000001",
		OS:          "linux", Arch: "amd64", RAMBytes: 16 << 30, CPU: "AMD Ryzen 7",
		CPUCores: 16, StorageBytes: 512 << 30, GPU: "llvmpipe", VulkanVersion: "1.3.255",
		AppVersion: "0.87.0", AddIns: []AddIn{{ID: "com.oblikovati.cam", Version: "0.6.0"}},
	}
}

func TestSubmitSendsCRCTokenOverExactBody(t *testing.T) {
	doer := &fakeDoer{status: http.StatusAccepted}
	sub := NewSubmitter("https://example.test/report", doer)
	if err := sub.Submit(context.Background(), sampleSnapshot()); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if doer.gotReq.Method != http.MethodPost {
		t.Errorf("method = %q, want POST", doer.gotReq.Method)
	}
	if ct := doer.gotReq.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q", ct)
	}
	want := fmt.Sprintf("%08x", crc32.ChecksumIEEE(doer.gotBody))
	if got := doer.gotReq.Header.Get("Authorization"); got != want {
		t.Errorf("authorization = %q, want %q (CRC of exact body)", got, want)
	}
}

func TestSubmitOfflineMapsToErrOffline(t *testing.T) {
	sub := NewSubmitter("", &fakeDoer{err: errors.New("dial tcp: no route to host")})
	if err := sub.Submit(context.Background(), sampleSnapshot()); !errors.Is(err, ErrOffline) {
		t.Fatalf("err = %v, want ErrOffline", err)
	}
}

func TestSubmitRejectsNon2xx(t *testing.T) {
	sub := NewSubmitter("", &fakeDoer{status: http.StatusUnauthorized, body: "invalid authorization token"})
	err := sub.Submit(context.Background(), sampleSnapshot())
	if err == nil || errors.Is(err, ErrOffline) {
		t.Fatalf("err = %v, want a non-offline error on 401", err)
	}
	if !strings.Contains(err.Error(), "invalid authorization token") {
		t.Errorf("err missing body context: %v", err)
	}
}

func TestNewSubmitterDefaultsEndpoint(t *testing.T) {
	doer := &fakeDoer{status: http.StatusOK}
	sub := NewSubmitter("", doer)
	if err := sub.Submit(context.Background(), sampleSnapshot()); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if doer.gotReq.URL.String() != DefaultEndpoint {
		t.Errorf("url = %q, want default %q", doer.gotReq.URL, DefaultEndpoint)
	}
}
