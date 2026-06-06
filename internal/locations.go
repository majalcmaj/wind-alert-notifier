package internal

import (
	_ "embed"
	"encoding/json"
)

type Location struct {
	ID   string  `json:"id"   dynamodbav:"id"`
	Name string  `json:"name" dynamodbav:"name"`
	Lat  float64 `json:"lat"  dynamodbav:"lat"`
	Lon  float64 `json:"lon"  dynamodbav:"lon"`
}

//go:embed locations.json
var locationsJSON []byte

var Locations []Location

func init() {
	if err := json.Unmarshal(locationsJSON, &Locations); err != nil {
		panic(err)
	}
}
