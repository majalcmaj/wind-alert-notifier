package internal

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type mockDynamoDB struct {
	scanOut  *dynamodb.ScanOutput
	queryOut *dynamodb.QueryOutput
}

func (m *mockDynamoDB) Scan(_ context.Context, _ *dynamodb.ScanInput, _ ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
	return m.scanOut, nil
}

func (m *mockDynamoDB) Query(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return m.queryOut, nil
}

func mustMarshalMap(v any) map[string]types.AttributeValue {
	m, err := attributevalue.MarshalMap(v)
	if err != nil {
		panic(err)
	}
	return m
}

func TestLoadLocations(t *testing.T) {
	want := Location{ID: "sopot", Name: "Sopot", Lat: 54.646034, Lon: 18.512407}
	mock := &mockDynamoDB{
		scanOut: &dynamodb.ScanOutput{
			Items: []map[string]types.AttributeValue{mustMarshalMap(want)},
		},
	}

	locs, err := LoadLocations(context.Background(), mock)
	if err != nil {
		t.Fatalf("LoadLocations error: %v", err)
	}
	if len(locs) != 1 {
		t.Fatalf("expected 1 location, got %d", len(locs))
	}
	got := locs[0]
	if got.ID != want.ID || got.Name != want.Name || got.Lat != want.Lat || got.Lon != want.Lon {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestLoadRulesForLocation(t *testing.T) {
	want := Rule{
		Name:       "Strong NW afternoon wind",
		LocationID: "sopot",
		AngleRange: Range{From: 270, To: 360},
		SpeedRange: Range{From: 6, To: 25},
		HourRange:  Range{From: 12, To: 20},
	}
	mock := &mockDynamoDB{
		queryOut: &dynamodb.QueryOutput{
			Items: []map[string]types.AttributeValue{mustMarshalMap(want)},
		},
	}

	rules, err := LoadRulesForLocation(context.Background(), mock, "sopot")
	if err != nil {
		t.Fatalf("LoadRulesForLocation error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	got := rules[0]
	if got.Name != want.Name || got.LocationID != want.LocationID {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if got.AngleRange != want.AngleRange || got.SpeedRange != want.SpeedRange || got.HourRange != want.HourRange {
		t.Errorf("ranges mismatch: got %+v, want %+v", got, want)
	}
}

func TestLoadLocationsEmpty(t *testing.T) {
	mock := &mockDynamoDB{scanOut: &dynamodb.ScanOutput{}}
	locs, err := LoadLocations(context.Background(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(locs) != 0 {
		t.Errorf("expected empty slice, got %d items", len(locs))
	}
}

func TestLoadRulesForLocationEmpty(t *testing.T) {
	mock := &mockDynamoDB{queryOut: &dynamodb.QueryOutput{}}
	rules, err := LoadRulesForLocation(context.Background(), mock, "noloc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected empty slice, got %d items", len(rules))
	}
}

// Ensure *dynamodb.Client satisfies the interface used by LoadLocations/LoadRulesForLocation.
var _ dynamoDBClient = (*dynamodb.Client)(nil)

// Ensure mock satisfies the same interface.
var _ dynamoDBClient = (*mockDynamoDB)(nil)
