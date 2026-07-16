package server

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"wind-alert/internal/model"
)

type fakeStore struct{}

func (f *fakeStore) LoadLocations(_ context.Context) ([]model.Location, error) { return nil, nil }
func (f *fakeStore) PutLocation(_ context.Context, _ model.Location) error     { return nil }
func (f *fakeStore) DeleteLocation(_ context.Context, _ string) error          { return nil }
func (f *fakeStore) LoadRulesForLocation(_ context.Context, _ string) ([]model.Rule, error) {
	return nil, nil
}
func (f *fakeStore) PutRule(_ context.Context, _ model.Rule) error            { return nil }
func (f *fakeStore) DeleteRule(_ context.Context, _, _ string) error          { return nil }
func (f *fakeStore) DeleteRulesForLocation(_ context.Context, _ string) error { return nil }

func basicAuthHeader(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}

func TestBasicAuthRejectsNoCredentials(t *testing.T) {
	t.Setenv("ADMIN_USER", "admin")
	t.Setenv("ADMIN_PASSWORD", "secret")

	srv := New(&fakeStore{})
	handler := BasicAuth(srv.Routes())

	req := httptest.NewRequest(http.MethodGet, "/locations", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	if w.Header().Get("WWW-Authenticate") == "" {
		t.Error("expected WWW-Authenticate header")
	}
}

func TestBasicAuthRejectsWrongCredentials(t *testing.T) {
	t.Setenv("ADMIN_USER", "admin")
	t.Setenv("ADMIN_PASSWORD", "secret")

	srv := New(&fakeStore{})
	handler := BasicAuth(srv.Routes())

	req := httptest.NewRequest(http.MethodGet, "/locations", nil)
	req.Header.Set("Authorization", basicAuthHeader("admin", "wrong"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestBasicAuthAcceptsCorrectCredentials(t *testing.T) {
	t.Setenv("ADMIN_USER", "admin")
	t.Setenv("ADMIN_PASSWORD", "secret")

	srv := New(&fakeStore{})
	handler := BasicAuth(srv.Routes())

	// use a rules route to verify auth passes through to a real handler
	req := httptest.NewRequest(http.MethodGet, "/locations/abc/rules", nil)
	req.Header.Set("Authorization", basicAuthHeader("admin", "secret"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestEmptyEnvFailsClosed(t *testing.T) {
	t.Setenv("ADMIN_USER", "")
	t.Setenv("ADMIN_PASSWORD", "")

	srv := New(&fakeStore{})
	handler := BasicAuth(srv.Routes())

	req := httptest.NewRequest(http.MethodGet, "/locations", nil)
	req.Header.Set("Authorization", basicAuthHeader("", ""))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when env empty, got %d", w.Code)
	}
}
