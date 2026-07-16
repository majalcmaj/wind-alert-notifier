package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wind-alert/web/internal/web"
)

func TestRender_layout(t *testing.T) {
	w := httptest.NewRecorder()
	if err := web.Render(w, "layout.html", nil); err != nil {
		t.Fatalf("Render: %v", err)
	}
	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "/static/pico.min.css") {
		t.Error("missing pico.min.css link")
	}
	if !strings.Contains(body, "/static/htmx.min.js") {
		t.Error("missing htmx.min.js script")
	}
}

func TestStaticHandler_js(t *testing.T) {
	h := web.StaticHandler()
	req := httptest.NewRequest("GET", "/htmx.min.js", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Errorf("Content-Type = %q, want javascript", ct)
	}
}

func TestStaticHandler_css(t *testing.T) {
	h := web.StaticHandler()
	req := httptest.NewRequest("GET", "/pico.min.css", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "css") {
		t.Errorf("Content-Type = %q, want css", ct)
	}
}
