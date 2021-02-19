package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPingHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/v1/minion/ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	s := server{}
	handler := http.HandlerFunc(s.PingHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := `pong`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestVersionHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/v1/minion/version", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	s := server{
		version: &apiVersion{
			Version:    "0.1.0",
			GitHash:    "No Git Commit Provided",
			BuildStamp: "No BuildStamp Provided",
		},
	}
	handler := http.HandlerFunc(s.VersionHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := `{"version":"0.1.0","githash":"No Git Commit Provided","buildstamp":"No BuildStamp Provided"}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}
