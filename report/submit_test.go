// SPDX-License-Identifier: GPL-2.0-only

package report

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

func samplePayload() Payload {
	return Payload{
		Comment:        "it crashed when I extruded",
		OS:             "linux",
		Arch:           "amd64",
		AppVersion:     "0.16.0",
		OpenDocuments:  []DocumentInfo{{Path: "/tmp/p.obk", Name: "p", Type: "Part"}},
		TransactionLog: []TransactionEvent{{Time: "09:00:01", Document: "p", Label: "Extrude", Recipe: "features: []\n"}},
		WindowPNG:      []byte{0x89, 0x50, 0x4e, 0x47},
		ViewportPNG:    []byte{0x89, 0x50, 0x4e, 0x47, 0x0d},
	}
}

func TestSubmitSendsCRCTokenOverExactBody(t *testing.T) {
	doer := &fakeDoer{status: http.StatusAccepted}
	sub := NewSubmitter("https://example.test/report", doer)

	if err := sub.Submit(context.Background(), samplePayload()); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if doer.gotReq.Method != http.MethodPost {
		t.Errorf("method = %q, want POST", doer.gotReq.Method)
	}
	if got := doer.gotReq.URL.String(); got != "https://example.test/report" {
		t.Errorf("url = %q", got)
	}
	if ct := doer.gotReq.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q", ct)
	}
	// The token must be the CRC-32 of the EXACT bytes that were sent — recompute over the
	// captured body and compare. This is the contract the server re-verifies.
	want := fmt.Sprintf("%08x", crc32.ChecksumIEEE(doer.gotBody))
	if got := doer.gotReq.Header.Get("Authorization"); got != want {
		t.Errorf("authorization = %q, want %q", got, want)
	}
}

func TestTokenIsDeterministicCRC32(t *testing.T) {
	body := []byte(`{"comment":"hi"}`)
	want := fmt.Sprintf("%08x", crc32.ChecksumIEEE(body))
	if got := Token(body); got != want {
		t.Errorf("Token = %q, want %q", got, want)
	}
	// Same content in a distinct slice must yield the same token (it hashes bytes, not identity).
	if a, b := Token(body), Token([]byte(`{"comment":"hi"}`)); a != b {
		t.Errorf("Token not content-deterministic: %q vs %q", a, b)
	}
}

func TestSubmitOfflineMapsToErrOffline(t *testing.T) {
	doer := &fakeDoer{err: errors.New("dial tcp: no route to host")}
	sub := NewSubmitter("", doer)
	err := sub.Submit(context.Background(), samplePayload())
	if !errors.Is(err, ErrOffline) {
		t.Fatalf("err = %v, want ErrOffline", err)
	}
}

func TestSubmitRejectsNon2xx(t *testing.T) {
	doer := &fakeDoer{status: http.StatusUnauthorized, body: "bad token"}
	sub := NewSubmitter("", doer)
	err := sub.Submit(context.Background(), samplePayload())
	if err == nil {
		t.Fatal("want error on 401")
	}
	if errors.Is(err, ErrOffline) {
		t.Error("401 should not be ErrOffline")
	}
	if !strings.Contains(err.Error(), "bad token") {
		t.Errorf("err missing body context: %v", err)
	}
}

func TestNewSubmitterDefaultsEndpoint(t *testing.T) {
	doer := &fakeDoer{status: http.StatusOK}
	sub := NewSubmitter("", doer)
	if err := sub.Submit(context.Background(), samplePayload()); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if doer.gotReq.URL.String() != DefaultEndpoint {
		t.Errorf("url = %q, want default %q", doer.gotReq.URL, DefaultEndpoint)
	}
}
