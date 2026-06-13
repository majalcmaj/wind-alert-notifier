package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/majalcmaj/wind-alert/shared/model"
)

// fakeLocationsStore wraps fakeStore and adds configurable locations + call recording.
type fakeLocationsStore struct {
	fakeStore
	locations        []model.Location
	putCalls         []model.Location
	deleteCalls      []string
	deleteRulesCalls []string
}

func (f *fakeLocationsStore) LoadLocations(_ context.Context) ([]model.Location, error) {
	return f.locations, nil
}

func (f *fakeLocationsStore) PutLocation(_ context.Context, loc model.Location) error {
	f.putCalls = append(f.putCalls, loc)
	return nil
}

func (f *fakeLocationsStore) DeleteLocation(_ context.Context, id string) error {
	f.deleteCalls = append(f.deleteCalls, id)
	return nil
}

func (f *fakeLocationsStore) DeleteRulesForLocation(_ context.Context, id string) error {
	f.deleteRulesCalls = append(f.deleteRulesCalls, id)
	return nil
}

func newLocationsHandler(t *testing.T, fs *fakeLocationsStore) http.Handler {
	t.Helper()
	t.Setenv("ADMIN_USER", "u")
	t.Setenv("ADMIN_PASSWORD", "p")
	return BasicAuth(New(fs).Routes())
}

func authedReq(method, path string, body url.Values) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, strings.NewReader(body.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Authorization", basicAuthHeader("u", "p"))
	return req
}

func TestLocationsPage(t *testing.T) {
	fs := &fakeLocationsStore{
		locations: []model.Location{
			{ID: "puck", Name: "Puck", Lat: 54.71, Lon: 18.40},
		},
	}
	h := newLocationsHandler(t, fs)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, authedReq(http.MethodGet, "/", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Puck") {
		t.Error("body should contain location name 'Puck'")
	}
}

func TestCreateLocationValid(t *testing.T) {
	fs := &fakeLocationsStore{
		locations: []model.Location{
			{ID: "puck", Name: "Puck", Lat: 54.71, Lon: 18.40},
		},
	}
	h := newLocationsHandler(t, fs)

	form := url.Values{
		"id":   {"sopot"},
		"name": {"Sopot"},
		"lat":  {"54.44"},
		"lon":  {"18.57"},
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, authedReq(http.MethodPost, "/locations", form))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if len(fs.putCalls) != 1 {
		t.Fatalf("expected 1 PutLocation call, got %d", len(fs.putCalls))
	}
	if fs.putCalls[0].ID != "sopot" {
		t.Errorf("unexpected put ID: %s", fs.putCalls[0].ID)
	}
	if !strings.Contains(w.Body.String(), "Puck") {
		t.Error("response should include refreshed list with existing location")
	}
}

func TestCreateLocationInvalidLat(t *testing.T) {
	fs := &fakeLocationsStore{}
	h := newLocationsHandler(t, fs)

	form := url.Values{
		"id":   {"bad"},
		"name": {"Bad"},
		"lat":  {"999"},
		"lon":  {"0"},
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, authedReq(http.MethodPost, "/locations", form))

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}
	if len(fs.putCalls) != 0 {
		t.Error("PutLocation must not be called on validation error")
	}
	if !strings.Contains(w.Body.String(), "lat") {
		t.Error("error message should mention lat")
	}
}

func TestEditLocationForm(t *testing.T) {
	fs := &fakeLocationsStore{
		locations: []model.Location{
			{ID: "puck", Name: "Puck", Lat: 54.71, Lon: 18.40},
		},
	}
	h := newLocationsHandler(t, fs)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, authedReq(http.MethodGet, "/locations/puck/edit", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Puck") {
		t.Error("edit form should contain location name")
	}
	if !strings.Contains(body, `value="puck"`) {
		t.Error("edit form should have id pre-filled")
	}
}

func TestEditLocationFormNotFound(t *testing.T) {
	fs := &fakeLocationsStore{}
	h := newLocationsHandler(t, fs)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, authedReq(http.MethodGet, "/locations/nope/edit", nil))

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUpdateLocation(t *testing.T) {
	fs := &fakeLocationsStore{
		locations: []model.Location{
			{ID: "puck", Name: "Puck", Lat: 54.71, Lon: 18.40},
		},
	}
	h := newLocationsHandler(t, fs)

	form := url.Values{
		"name": {"Puck Updated"},
		"lat":  {"54.72"},
		"lon":  {"18.41"},
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, authedReq(http.MethodPut, "/locations/puck", form))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if len(fs.putCalls) != 1 {
		t.Fatalf("expected 1 PutLocation call, got %d", len(fs.putCalls))
	}
	if fs.putCalls[0].Name != "Puck Updated" {
		t.Errorf("unexpected updated name: %s", fs.putCalls[0].Name)
	}
	if !strings.Contains(w.Body.String(), "Puck Updated") {
		t.Error("response row should contain updated name")
	}
}

func TestDeleteLocationCascade(t *testing.T) {
	fs := &fakeLocationsStore{
		locations: []model.Location{
			{ID: "puck", Name: "Puck", Lat: 54.71, Lon: 18.40},
		},
	}
	h := newLocationsHandler(t, fs)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, authedReq(http.MethodDelete, "/locations/puck", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if len(fs.deleteRulesCalls) != 1 || fs.deleteRulesCalls[0] != "puck" {
		t.Error("DeleteRulesForLocation must be called with 'puck'")
	}
	if len(fs.deleteCalls) != 1 || fs.deleteCalls[0] != "puck" {
		t.Error("DeleteLocation must be called with 'puck'")
	}
}

func TestDeleteLocationNotFound(t *testing.T) {
	fs := &fakeLocationsStore{}
	h := newLocationsHandler(t, fs)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, authedReq(http.MethodDelete, "/locations/nope", nil))

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	if len(fs.deleteCalls) != 0 {
		t.Error("DeleteLocation must not be called for unknown id")
	}
}
