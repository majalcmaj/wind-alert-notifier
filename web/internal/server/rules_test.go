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

type ruleDeleteCall struct {
	locationID string
	name       string
}

// fakeRulesStore wraps fakeStore and adds configurable locations + rules,
// recording every PutRule/DeleteRule call (and their relative order).
type fakeRulesStore struct {
	fakeStore
	locations       []model.Location
	rules           []model.Rule
	putRuleCalls    []model.Rule
	deleteRuleCalls []ruleDeleteCall
	callOrder       []string
}

func (f *fakeRulesStore) LoadLocations(_ context.Context) ([]model.Location, error) {
	return f.locations, nil
}

func (f *fakeRulesStore) LoadRulesForLocation(_ context.Context, id string) ([]model.Rule, error) {
	var out []model.Rule
	for _, r := range f.rules {
		if r.LocationID == id {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeRulesStore) PutRule(_ context.Context, rule model.Rule) error {
	f.putRuleCalls = append(f.putRuleCalls, rule)
	f.callOrder = append(f.callOrder, "put:"+rule.Name)

	for i, existing := range f.rules {
		if existing.LocationID == rule.LocationID && existing.Name == rule.Name {
			f.rules[i] = rule
			return nil
		}
	}
	f.rules = append(f.rules, rule)
	return nil
}

func (f *fakeRulesStore) DeleteRule(_ context.Context, locationID, name string) error {
	f.deleteRuleCalls = append(f.deleteRuleCalls, ruleDeleteCall{locationID: locationID, name: name})
	f.callOrder = append(f.callOrder, "delete:"+name)

	for i, r := range f.rules {
		if r.LocationID == locationID && r.Name == name {
			f.rules = append(f.rules[:i], f.rules[i+1:]...)
			break
		}
	}
	return nil
}

func newRulesHandler(t *testing.T, fs *fakeRulesStore) http.Handler {
	t.Helper()
	t.Setenv("ADMIN_USER", "u")
	t.Setenv("ADMIN_PASSWORD", "p")
	return BasicAuth(New(fs).Routes())
}

// ruleForm returns a valid base rule form, with overrides applied on top.
func ruleForm(overrides url.Values) url.Values {
	form := url.Values{
		"name":           {"strong-onshore"},
		"angle_from":     {"0"},
		"angle_to":       {"90"},
		"speed_from":     {"5"},
		"speed_to":       {"15"},
		"hour_from":      {"6"},
		"hour_to":        {"20"},
		"min_confidence": {"0.5"},
	}
	for k, v := range overrides {
		form[k] = v
	}
	return form
}

func TestListRules(t *testing.T) {
	fs := &fakeRulesStore{
		locations: []model.Location{{ID: "puck", Name: "Puck"}},
		rules: []model.Rule{
			{LocationID: "puck", Name: "onshore", AngleRange: model.Range{From: 0, To: 90}},
		},
	}
	h := newRulesHandler(t, fs)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, authedReq(http.MethodGet, "/locations/puck/rules", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "onshore") {
		t.Error("response should list rule 'onshore'")
	}
}

func TestCreateRuleValidCyclicAngle(t *testing.T) {
	fs := &fakeRulesStore{
		locations: []model.Location{{ID: "puck", Name: "Puck"}},
	}
	h := newRulesHandler(t, fs)

	form := ruleForm(url.Values{
		"name":       {"wrap-rule"},
		"angle_from": {"350"},
		"angle_to":   {"10"},
	})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, authedReq(http.MethodPost, "/locations/puck/rules", form))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if len(fs.putRuleCalls) != 1 {
		t.Fatalf("expected 1 PutRule call, got %d", len(fs.putRuleCalls))
	}
	if fs.putRuleCalls[0].AngleRange.From != 350 || fs.putRuleCalls[0].AngleRange.To != 10 {
		t.Errorf("cyclic angle range not preserved: %+v", fs.putRuleCalls[0].AngleRange)
	}
	if !strings.Contains(w.Body.String(), "wrap-rule") {
		t.Error("response should include the new rule")
	}
}

func TestCreateRuleInvalidSpeedRange(t *testing.T) {
	fs := &fakeRulesStore{
		locations: []model.Location{{ID: "puck", Name: "Puck"}},
	}
	h := newRulesHandler(t, fs)

	form := ruleForm(url.Values{
		"speed_from": {"20"},
		"speed_to":   {"5"},
	})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, authedReq(http.MethodPost, "/locations/puck/rules", form))

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}
	if len(fs.putRuleCalls) != 0 {
		t.Error("PutRule must not be called on validation error")
	}
	if !strings.Contains(w.Body.String(), "speed") {
		t.Error("error message should mention speed")
	}
}

func TestCreateRuleMissingLocation(t *testing.T) {
	fs := &fakeRulesStore{}
	h := newRulesHandler(t, fs)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, authedReq(http.MethodPost, "/locations/nope/rules", ruleForm(nil)))

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	if len(fs.putRuleCalls) != 0 {
		t.Error("PutRule must not be called for a missing location")
	}
}

func TestEditRuleForm(t *testing.T) {
	fs := &fakeRulesStore{
		locations: []model.Location{{ID: "puck", Name: "Puck"}},
		rules: []model.Rule{
			{LocationID: "puck", Name: "onshore", AngleRange: model.Range{From: 0, To: 90}, MinConfidence: 0.7},
		},
	}
	h := newRulesHandler(t, fs)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, authedReq(http.MethodGet, "/locations/puck/rules/onshore/edit", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `value="onshore"`) {
		t.Error("form should be pre-filled with the rule's name")
	}
	if !strings.Contains(body, `value="0.70"`) {
		t.Error("form should be pre-filled with min_confidence")
	}
}

func TestEditRuleFormNotFound(t *testing.T) {
	fs := &fakeRulesStore{locations: []model.Location{{ID: "puck", Name: "Puck"}}}
	h := newRulesHandler(t, fs)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, authedReq(http.MethodGet, "/locations/puck/rules/nope/edit", nil))

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUpdateRuleNoRename(t *testing.T) {
	fs := &fakeRulesStore{
		locations: []model.Location{{ID: "puck", Name: "Puck"}},
		rules: []model.Rule{
			{LocationID: "puck", Name: "onshore", AngleRange: model.Range{From: 0, To: 90}},
		},
	}
	h := newRulesHandler(t, fs)

	form := ruleForm(url.Values{"name": {"onshore"}})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, authedReq(http.MethodPut, "/locations/puck/rules/onshore", form))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if len(fs.putRuleCalls) != 1 {
		t.Fatalf("expected 1 PutRule call, got %d", len(fs.putRuleCalls))
	}
	if len(fs.deleteRuleCalls) != 0 {
		t.Error("DeleteRule must not be called when the name is unchanged")
	}
}

func TestUpdateRuleInvalid(t *testing.T) {
	fs := &fakeRulesStore{
		locations: []model.Location{{ID: "puck", Name: "Puck"}},
		rules: []model.Rule{
			{LocationID: "puck", Name: "onshore", AngleRange: model.Range{From: 0, To: 90}},
		},
	}
	h := newRulesHandler(t, fs)

	form := ruleForm(url.Values{
		"name":       {"onshore"},
		"speed_from": {"20"},
		"speed_to":   {"5"},
	})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, authedReq(http.MethodPut, "/locations/puck/rules/onshore", form))

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}
	if len(fs.putRuleCalls) != 0 || len(fs.deleteRuleCalls) != 0 {
		t.Error("invalid update must not write")
	}
}

func TestUpdateRuleRename(t *testing.T) {
	fs := &fakeRulesStore{
		locations: []model.Location{{ID: "puck", Name: "Puck"}},
		rules: []model.Rule{
			{LocationID: "puck", Name: "old-name", AngleRange: model.Range{From: 0, To: 90}},
		},
	}
	h := newRulesHandler(t, fs)

	form := ruleForm(url.Values{"name": {"new-name"}})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, authedReq(http.MethodPut, "/locations/puck/rules/old-name", form))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if len(fs.putRuleCalls) != 1 || fs.putRuleCalls[0].Name != "new-name" {
		t.Fatalf("expected PutRule(new-name), got %+v", fs.putRuleCalls)
	}
	if len(fs.deleteRuleCalls) != 1 || fs.deleteRuleCalls[0].name != "old-name" {
		t.Fatalf("expected DeleteRule(old-name), got %+v", fs.deleteRuleCalls)
	}
	if len(fs.callOrder) != 2 || fs.callOrder[0] != "put:new-name" || fs.callOrder[1] != "delete:old-name" {
		t.Errorf("expected put-before-delete order, got %v", fs.callOrder)
	}
	if !strings.Contains(w.Body.String(), "new-name") {
		t.Error("response should show the renamed rule")
	}
}

func TestDeleteRule(t *testing.T) {
	fs := &fakeRulesStore{
		locations: []model.Location{{ID: "puck", Name: "Puck"}},
		rules: []model.Rule{
			{LocationID: "puck", Name: "onshore", AngleRange: model.Range{From: 0, To: 90}},
		},
	}
	h := newRulesHandler(t, fs)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, authedReq(http.MethodDelete, "/locations/puck/rules/onshore", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if len(fs.deleteRuleCalls) != 1 || fs.deleteRuleCalls[0].name != "onshore" {
		t.Fatalf("expected DeleteRule(onshore), got %+v", fs.deleteRuleCalls)
	}
	if strings.Contains(w.Body.String(), "onshore") {
		t.Error("deleted rule should not appear in the refreshed list")
	}
}
