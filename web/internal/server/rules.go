package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/majalcmaj/wind-alert/shared/model"
	"github.com/majalcmaj/wind-alert/web/internal/validate"
)

type rulesListData struct {
	LocationID string
	Rules      []model.Rule
}

type ruleFormData struct {
	LocationID string
	OldName    string
	Rule       model.Rule
	Error      string
	IsEdit     bool
}

func locationExists(locs []model.Location, id string) bool {
	for _, l := range locs {
		if l.ID == id {
			return true
		}
	}
	return false
}

// parseRuleForm reads the rule fields (name + the six range floats +
// min_confidence) from a submitted form.
func parseRuleForm(r *http.Request, locationID string) (model.Rule, error) {
	rule := model.Rule{LocationID: locationID, Name: r.FormValue("name")}

	floatFields := []struct {
		name string
		dst  *float64
	}{
		{"angle_from", &rule.AngleRange.From},
		{"angle_to", &rule.AngleRange.To},
		{"speed_from", &rule.SpeedRange.From},
		{"speed_to", &rule.SpeedRange.To},
		{"hour_from", &rule.HourRange.From},
		{"hour_to", &rule.HourRange.To},
		{"min_confidence", &rule.MinConfidence},
	}
	for _, f := range floatFields {
		v, err := strconv.ParseFloat(r.FormValue(f.name), 64)
		if err != nil {
			return rule, fmt.Errorf("%s: must be a valid number", f.name)
		}
		*f.dst = v
	}
	return rule, nil
}

func (s *Server) listRules(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rules, err := s.ds.LoadRulesForLocation(r.Context(), id)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to load rules")
		return
	}
	render(w, "rules_list", rulesListData{LocationID: id, Rules: rules})
}

func (s *Server) newRuleForm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	render(w, "rule_form", ruleFormData{LocationID: id, Rule: model.Rule{LocationID: id}})
}

func (s *Server) createRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		httpError(w, http.StatusBadRequest, "bad form data")
		return
	}

	rule, err := parseRuleForm(r, id)
	if err != nil {
		s.renderRuleFormError(w, ruleFormData{LocationID: id, Rule: rule, Error: err.Error()})
		return
	}

	locs, err := s.ds.LoadLocations(r.Context())
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to load locations")
		return
	}
	if !locationExists(locs, id) {
		httpError(w, http.StatusNotFound, "location not found")
		return
	}

	if err := validate.ValidateRule(rule); err != nil {
		s.renderRuleFormError(w, ruleFormData{LocationID: id, Rule: rule, Error: err.Error()})
		return
	}

	if err := s.ds.PutRule(r.Context(), rule); err != nil {
		httpError(w, http.StatusInternalServerError, "failed to create rule")
		return
	}

	s.renderRulesList(w, r, id)
}

func (s *Server) editRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	name := r.PathValue("name")

	rules, err := s.ds.LoadRulesForLocation(r.Context(), id)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to load rules")
		return
	}
	for _, rule := range rules {
		if rule.Name == name {
			render(w, "rule_form", ruleFormData{LocationID: id, OldName: name, Rule: rule, IsEdit: true})
			return
		}
	}
	httpError(w, http.StatusNotFound, "rule not found")
}

func (s *Server) updateRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	oldName := r.PathValue("name")

	if err := r.ParseForm(); err != nil {
		httpError(w, http.StatusBadRequest, "bad form data")
		return
	}

	rule, err := parseRuleForm(r, id)
	if err != nil {
		s.renderRuleFormError(w, ruleFormData{LocationID: id, OldName: oldName, Rule: rule, Error: err.Error(), IsEdit: true})
		return
	}

	if err := validate.ValidateRule(rule); err != nil {
		s.renderRuleFormError(w, ruleFormData{LocationID: id, OldName: oldName, Rule: rule, Error: err.Error(), IsEdit: true})
		return
	}

	if err := s.ds.PutRule(r.Context(), rule); err != nil {
		httpError(w, http.StatusInternalServerError, "failed to update rule")
		return
	}

	if rule.Name != oldName {
		if err := s.ds.DeleteRule(r.Context(), id, oldName); err != nil {
			httpError(w, http.StatusInternalServerError, "failed to delete old rule")
			return
		}
	}

	s.renderRulesList(w, r, id)
}

func (s *Server) deleteRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	name := r.PathValue("name")

	if err := s.ds.DeleteRule(r.Context(), id, name); err != nil {
		httpError(w, http.StatusInternalServerError, "failed to delete rule")
		return
	}

	s.renderRulesList(w, r, id)
}

func (s *Server) renderRuleFormError(w http.ResponseWriter, fd ruleFormData) {
	w.Header().Set("HX-Retarget", "#rule-form-"+fd.LocationID)
	w.Header().Set("HX-Reswap", "innerHTML")
	renderStatus(w, http.StatusUnprocessableEntity, "rule_form", fd)
}

func (s *Server) renderRulesList(w http.ResponseWriter, r *http.Request, locationID string) {
	rules, err := s.ds.LoadRulesForLocation(r.Context(), locationID)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed to reload rules")
		return
	}
	render(w, "rules_list", rulesListData{LocationID: locationID, Rules: rules})
}
