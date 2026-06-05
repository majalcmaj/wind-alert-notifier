package internal

import (
	_ "embed"
	"encoding/json"
)

//go:embed locations.json
var locationsJSON []byte

var Locations []Location

func init() {
	if err := json.Unmarshal(locationsJSON, &Locations); err != nil {
		panic(err)
	}
}
