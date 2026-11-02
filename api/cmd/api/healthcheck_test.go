package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRunHealthcheck(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer ok.Close()
	if err := runHealthcheck(ok.Client(), ok.URL); err != nil {
		t.Errorf("200 should be healthy, got %v", err)
	}

	for _, code := range []int{http.StatusServiceUnavailable, http.StatusInternalServerError, http.StatusNotFound, http.StatusNoContent} {
		bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(code) }))
		if err := runHealthcheck(bad.Client(), bad.URL); err == nil {
			t.Errorf("status %d must be unhealthy", code)
		}
		bad.Close()
	}
}

func TestRunHealthcheckFailsWhenNothingIsListening(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // the port is now closed
	if err := runHealthcheck(&http.Client{Timeout: time.Second}, url); err == nil {
		t.Error("a refused connection must be unhealthy")
	}
}

func TestRunHealthcheckFailsOnTimeout(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer slow.Close()
	if err := runHealthcheck(&http.Client{Timeout: 50 * time.Millisecond}, slow.URL); err == nil {
		t.Error("a hung server must be unhealthy")
	}
}

func TestHealthClientHasATimeout(t *testing.T) {
	if healthClient().Timeout <= 0 {
		t.Error("the healthcheck client must never wait forever")
	}
}
