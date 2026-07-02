// SPDX-License-Identifier: GPL-2.0-only

package crcpost

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTokenIsDeterministicCRC32(t *testing.T) {
	body := []byte(`{"k":"v"}`)
	if Token(body) != Token([]byte(`{"k":"v"}`)) {
		t.Error("Token not content-deterministic")
	}
	if len(Token(body)) != 8 {
		t.Errorf("token %q is not 8 hex chars", Token(body))
	}
}

func TestSendPostsJSONWithCRCTokenOverExactBody(t *testing.T) {
	var gotAuth, gotCT, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	body := []byte(`{"machineUUID":"x"}`)
	if err := Send(context.Background(), srv.Client(), srv.URL, body); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotCT != "application/json" {
		t.Errorf("content-type = %q", gotCT)
	}
	if gotBody != string(body) {
		t.Errorf("server saw body %q, want %q", gotBody, body)
	}
	if gotAuth != Token(body) {
		t.Errorf("authorization = %q, want CRC token %q", gotAuth, Token(body))
	}
}

func TestSendAccepts200And202(t *testing.T) {
	for _, code := range []int{http.StatusOK, http.StatusAccepted} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(code)
		}))
		if err := Send(context.Background(), srv.Client(), srv.URL, []byte("{}")); err != nil {
			t.Errorf("status %d: Send returned %v", code, err)
		}
		srv.Close()
	}
}

func TestSendRejectsNon2xxWithBodyContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, "invalid authorization token")
	}))
	defer srv.Close()
	err := Send(context.Background(), srv.Client(), srv.URL, []byte("{}"))
	if err == nil || errors.Is(err, ErrOffline) {
		t.Fatalf("err = %v, want a non-offline error on 401", err)
	}
	if !strings.Contains(err.Error(), "invalid authorization token") {
		t.Errorf("err missing body context: %v", err)
	}
}

func TestSendOfflineMapsToErrOffline(t *testing.T) {
	// A closed server's URL yields a connection-refused transport error.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	client := srv.Client()
	srv.Close()
	if err := Send(context.Background(), client, url, []byte("{}")); !errors.Is(err, ErrOffline) {
		t.Fatalf("err = %v, want ErrOffline", err)
	}
}

func TestSendBadEndpointErrors(t *testing.T) {
	// A control character in the URL makes NewRequestWithContext fail before any transport.
	err := Send(context.Background(), http.DefaultClient, "http://\x7f bad", []byte("{}"))
	if err == nil || errors.Is(err, ErrOffline) {
		t.Fatalf("err = %v, want a build-request error (not offline)", err)
	}
}
