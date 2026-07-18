package server

import (
	"log"
	"net/http"
	"strconv"

	"wind-alert/internal/model"
	"wind-alert/web/internal/validate"
)

type locationFormData struct {
	Location model.Location
	Error    string
	IsEdit   bool
}

func (s *Server) locationsPage(w http.ResponseWriter, r *http.Request) {
	locs, err := s.ds.LoadLocations(r.Context())
	if err != nil {
		log.Printf("load locations: %v", err)
		httpError(w, http.StatusInternalServerError, "failed to load locations")
		return
	}
	render(w, "layout.html", locs)
}

func (s *Server) locationsList(w http.ResponseWriter, r *http.Request) {
	locs, err := s.ds.LoadLocations(r.Context())
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to load locations")
		return
	}
	render(w, "locations_list", locs)
}

func (s *Server) newLocationForm(w http.ResponseWriter, r *http.Request) {
	render(w, "location_form", locationFormData{})
}

func (s *Server) createLocation(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, http.StatusBadRequest, "bad form data")
		return
	}

	lat, latErr := strconv.ParseFloat(r.FormValue("lat"), 64)
	lon, lonErr := strconv.ParseFloat(r.FormValue("lon"), 64)

	loc := model.Location{
		ID:   r.FormValue("id"),
		Name: r.FormValue("name"),
	}

	if latErr != nil || lonErr != nil {
		fd := locationFormData{Location: loc, Error: "lat and lon must be valid numbers"}
		w.Header().Set("HX-Retarget", "#location-form")
		w.Header().Set("HX-Reswap", "innerHTML")
		renderStatus(w, http.StatusUnprocessableEntity, "location_form", fd)
		return
	}
	loc.Lat = lat
	loc.Lon = lon

	if err := validate.ValidateLocation(loc); err != nil {
		fd := locationFormData{Location: loc, Error: err.Error()}
		w.Header().Set("HX-Retarget", "#location-form")
		w.Header().Set("HX-Reswap", "innerHTML")
		renderStatus(w, http.StatusUnprocessableEntity, "location_form", fd)
		return
	}

	if err := s.ds.PutLocation(r.Context(), loc); err != nil {
		httpError(w, http.StatusInternalServerError, "failed to create location")
		return
	}

	locs, err := s.ds.LoadLocations(r.Context())
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to reload locations")
		return
	}
	render(w, "locations_list", locs)
}

func (s *Server) editLocationForm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	locs, err := s.ds.LoadLocations(r.Context())
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to load locations")
		return
	}
	for _, l := range locs {
		if l.ID == id {
			render(w, "location_form", locationFormData{Location: l, IsEdit: true})
			return
		}
	}
	httpError(w, http.StatusNotFound, "location not found")
}

func (s *Server) updateLocation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		httpError(w, http.StatusBadRequest, "bad form data")
		return
	}

	lat, latErr := strconv.ParseFloat(r.FormValue("lat"), 64)
	lon, lonErr := strconv.ParseFloat(r.FormValue("lon"), 64)

	loc := model.Location{
		ID:   id, // use path value — ignore any posted id
		Name: r.FormValue("name"),
	}

	if latErr != nil || lonErr != nil {
		fd := locationFormData{Location: loc, Error: "lat and lon must be valid numbers", IsEdit: true}
		w.Header().Set("HX-Retarget", "#location-form")
		w.Header().Set("HX-Reswap", "innerHTML")
		renderStatus(w, http.StatusUnprocessableEntity, "location_form", fd)
		return
	}
	loc.Lat = lat
	loc.Lon = lon

	if err := validate.ValidateLocation(loc); err != nil {
		fd := locationFormData{Location: loc, Error: err.Error(), IsEdit: true}
		w.Header().Set("HX-Retarget", "#location-form")
		w.Header().Set("HX-Reswap", "innerHTML")
		renderStatus(w, http.StatusUnprocessableEntity, "location_form", fd)
		return
	}

	if err := s.ds.PutLocation(r.Context(), loc); err != nil {
		httpError(w, http.StatusInternalServerError, "failed to update location")
		return
	}

	render(w, "location_row", loc)
}

func (s *Server) deleteLocation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	locs, err := s.ds.LoadLocations(r.Context())
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to load locations")
		return
	}
	found := false
	for _, l := range locs {
		if l.ID == id {
			found = true
			break
		}
	}
	if !found {
		httpError(w, http.StatusNotFound, "location not found")
		return
	}

	if err := s.ds.DeleteRulesForLocation(r.Context(), id); err != nil {
		httpError(w, http.StatusInternalServerError, "failed to delete rules")
		return
	}
	if err := s.ds.DeleteLocation(r.Context(), id); err != nil {
		httpError(w, http.StatusInternalServerError, "failed to delete location")
		return
	}
	w.WriteHeader(http.StatusOK)
}
