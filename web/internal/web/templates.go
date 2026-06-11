package web

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
)

//go:embed templates/*.html
var tmplFS embed.FS

var tmpl = template.Must(
	template.New("").Funcs(funcs).ParseFS(tmplFS, "templates/*.html"),
)

var funcs = template.FuncMap{
	"fmtFloat": func(f float64) string { return fmt.Sprintf("%.2f", f) },
}

func Render(w http.ResponseWriter, name string, data any) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return execute(w, name, data)
}

func RenderStatus(w http.ResponseWriter, status int, name string, data any) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	return execute(w, name, data)
}

func execute(w http.ResponseWriter, name string, data any) error {
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render %s: %v", name, err)
		return err
	}
	return nil
}
