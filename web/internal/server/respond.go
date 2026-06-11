package server

import (
	"fmt"
	"net/http"

	"github.com/majalcmaj/wind-alert/web/internal/web"
)

func httpError(w http.ResponseWriter, status int, msg string) {
	http.Error(w, fmt.Sprintf("%d %s", status, msg), status)
}

func render(w http.ResponseWriter, name string, data any) {
	if err := web.Render(w, name, data); err != nil {
		httpError(w, http.StatusInternalServerError, "internal server error")
	}
}

func renderStatus(w http.ResponseWriter, status int, name string, data any) {
	web.RenderStatus(w, status, name, data) //nolint:errcheck
}
