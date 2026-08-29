package remote

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRemoteManagerUsesResolvedDestinationWithoutAmbientProxy(t *testing.T) {
	var requests atomic.Int32
	var wrongPath atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/api/remote/session" {
			wrongPath.Store(true)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"success":true,"sessionId":"session-1","secretToken":"secret-1"}`)
	}))
	defer server.Close()

	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")

	rm := NewRemoteManager(strings.Replace(server.URL, "127.0.0.1", "localhost", 1), "api-key")
	rm.Policy.AllowLocalDevelopment = true
	defer rm.Close()
	session, err := rm.InitSession(context.Background(), "workspace")
	if err != nil {
		t.Fatalf("InitSession() error = %v", err)
	}
	if session.SessionID != "session-1" || requests.Load() != 1 {
		t.Fatalf("unexpected session/request count: %#v, %d", session, requests.Load())
	}
	if wrongPath.Load() {
		t.Fatal("remote request used an unexpected path")
	}

	transport, ok := rm.HTTPClient.Transport.(*remoteTransport)
	if !ok || transport.base.Proxy != nil {
		t.Fatalf("remote client is not using a direct policy transport: %#v", rm.HTTPClient.Transport)
	}
}

func TestRemoteManagerRejectsCrossOriginRedirect(t *testing.T) {
	var redirected atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirected.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/api/remote/session", http.StatusTemporaryRedirect)
	}))
	defer source.Close()

	rm := NewRemoteManager(source.URL, "api-key")
	rm.Policy.AllowLocalDevelopment = true
	_, err := rm.InitSession(context.Background(), "workspace")
	if err == nil || !strings.Contains(err.Error(), "remote redirect changes origin") {
		t.Fatalf("InitSession() error = %v, want cross-origin redirect rejection", err)
	}
	if redirected.Load() != 0 {
		t.Fatalf("cross-origin redirect reached target %d times", redirected.Load())
	}
}

func TestRemotePolicyRejectsNumericLoopback(t *testing.T) {
	if _, err := (RemotePolicy{}).Validate("http://2130706433:8080"); err == nil {
		t.Fatal("Validate() accepted numeric loopback destination")
	}
}

func TestRemoteManagerCloseIsIdempotent(t *testing.T) {
	rm := NewRemoteManager("https://example.com", "api-key")
	rm.IsActive = true
	rm.Close()
	rm.Close()
	if rm.IsActive {
		t.Fatal("Close() left manager active")
	}
}
