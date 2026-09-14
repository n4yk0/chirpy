package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerReadiness(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		rec := httptest.NewRecorder()
		handlerReadiness(rec, httptest.NewRequest(method, "/healthz", nil))

		if rec.Code != http.StatusOK {
			t.Errorf("%s /healthz: status = %d, want %d", method, rec.Code, http.StatusOK)
		}
		if got, want := rec.Header().Get("Content-Type"), "text/plain; charset=utf-8"; got != want {
			t.Errorf("%s /healthz: Content-Type = %q, want %q", method, got, want)
		}
		if got, want := rec.Body.String(), "OK"; got != want {
			t.Errorf("%s /healthz: body = %q, want %q", method, got, want)
		}
	}
}

func TestFileserverRoutes(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot))))

	tests := []struct {
		path       string
		wantStatus int
	}{
		{"/app/", http.StatusOK},
		{"/app/index.html", http.StatusMovedPermanently}, // FileServer redirects /index.html to ./
		{"/app/assets/logo.png", http.StatusOK},
		{"/app/doesntexist.html", http.StatusNotFound},
		{"/index.html", http.StatusNotFound},
	}

	for _, tt := range tests {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
		if rec.Code != tt.wantStatus {
			t.Errorf("GET %s: status = %d, want %d", tt.path, rec.Code, tt.wantStatus)
		}
	}
}
