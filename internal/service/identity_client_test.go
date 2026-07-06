package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPIdentityClient_GetUserName(t *testing.T) {
	t.Run("returns the name for a known user", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/internal/users/7" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":7,"name":"Arthur"}`))
		}))
		defer srv.Close()

		client := NewHTTPIdentityClient(srv.URL)
		name, err := client.GetUserName(context.Background(), 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if name != "Arthur" {
			t.Fatalf("expected name %q, got %q", "Arthur", name)
		}
	})

	t.Run("returns an error on non-200 status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		client := NewHTTPIdentityClient(srv.URL)
		if _, err := client.GetUserName(context.Background(), 99); err == nil {
			t.Fatal("expected an error for a 404 response, got nil")
		}
	})
}
