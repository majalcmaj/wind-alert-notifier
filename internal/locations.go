package internal

import (
	_ "embed"
	"encoding/json"
)

type Location struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
}

//go:embed locations.json
var locationsJSON []byte

var Locations []Location

func init() {
	if err := json.Unmarshal(locationsJSON, &Locations); err != nil {
		panic(err)
	}
}
