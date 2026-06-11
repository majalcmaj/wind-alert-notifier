package dynamo

import (
	"context"
	"os"
	"testing"

	"github.com/majalcmaj/wind-alert/shared/model"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	ep := os.Getenv("DYNAMODB_ENDPOINT")
	if ep == "" {
		t.Skip("set DYNAMODB_ENDPOINT to run store tests")
	}
	ctx := context.Background()
	s, err := New(ctx)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

func TestLocationRoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	loc := model.Location{ID: "test-loc", Name: "Test Location", Lat: 46.8, Lon: 17.7}

	if err := s.PutLocation(ctx, loc); err != nil {
		t.Fatalf("PutLocation: %v", err)
	}

	locs, err := s.LoadLocations(ctx)
	if err != nil {
		t.Fatalf("LoadLocations: %v", err)
	}
	found := false
	for _, l := range locs {
		if l.ID == loc.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("PutLocation: item not found after put")
	}

	if err := s.DeleteLocation(ctx, loc.ID); err != nil {
		t.Fatalf("DeleteLocation: %v", err)
	}

	locs, err = s.LoadLocations(ctx)
	if err != nil {
		t.Fatalf("LoadLocations after delete: %v", err)
	}
	for _, l := range locs {
		if l.ID == loc.ID {
			t.Error("DeleteLocation: item still present after delete")
		}
	}
}

func TestDeleteRulesForLocation(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	locID := "test-cascade-loc"
	rules := []model.Rule{
		{LocationID: locID, Name: "rule-1", SpeedRange: model.Range{From: 0, To: 10}},
		{LocationID: locID, Name: "rule-2", SpeedRange: model.Range{From: 10, To: 20}},
		{LocationID: locID, Name: "rule-3", SpeedRange: model.Range{From: 20, To: 30}},
	}

	for _, r := range rules {
		if err := s.PutRule(ctx, r); err != nil {
			t.Fatalf("PutRule %s: %v", r.Name, err)
		}
	}

	loaded, err := s.LoadRulesForLocation(ctx, locID)
	if err != nil {
		t.Fatalf("LoadRulesForLocation: %v", err)
	}
	if len(loaded) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(loaded))
	}

	if err := s.DeleteRulesForLocation(ctx, locID); err != nil {
		t.Fatalf("DeleteRulesForLocation: %v", err)
	}

	loaded, err = s.LoadRulesForLocation(ctx, locID)
	if err != nil {
		t.Fatalf("LoadRulesForLocation after cascade: %v", err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected 0 rules after cascade delete, got %d", len(loaded))
	}

	// no-op when empty
	if err := s.DeleteRulesForLocation(ctx, locID); err != nil {
		t.Fatalf("DeleteRulesForLocation no-op: %v", err)
	}
}
