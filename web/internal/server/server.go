package server

import (
	"context"
	"net/http"

	"github.com/majalcmaj/wind-alert/shared/model"
	"github.com/majalcmaj/wind-alert/web/internal/web"
)

type Datastore interface {
	LoadLocations(ctx context.Context) ([]model.Location, error)
	PutLocation(ctx context.Context, loc model.Location) error
	DeleteLocation(ctx context.Context, id string) error
	LoadRulesForLocation(ctx context.Context, id string) ([]model.Rule, error)
	PutRule(ctx context.Context, rule model.Rule) error
	DeleteRule(ctx context.Context, id, name string) error
	DeleteRulesForLocation(ctx context.Context, id string) error
}

type Server struct {
	ds Datastore
}

func New(ds Datastore) *Server {
	return &Server{ds: ds}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", s.locationsPage)
	mux.Handle("GET /static/{file}", http.StripPrefix("/static/", web.StaticHandler()))

	mux.HandleFunc("GET /locations", s.locationsList)
	mux.HandleFunc("GET /locations/new", s.newLocationForm)
	mux.HandleFunc("POST /locations", s.createLocation)
	mux.HandleFunc("GET /locations/{id}/edit", s.editLocationForm)
	mux.HandleFunc("PUT /locations/{id}", s.updateLocation)
	mux.HandleFunc("DELETE /locations/{id}", s.deleteLocation)

	mux.HandleFunc("GET /locations/{id}/rules", s.listRules)
	mux.HandleFunc("GET /locations/{id}/rules/new", s.newRuleForm)
	mux.HandleFunc("POST /locations/{id}/rules", s.createRule)
	mux.HandleFunc("GET /locations/{id}/rules/{name}/edit", s.editRule)
	mux.HandleFunc("PUT /locations/{id}/rules/{name}", s.updateRule)
	mux.HandleFunc("DELETE /locations/{id}/rules/{name}", s.deleteRule)

	return mux
}
