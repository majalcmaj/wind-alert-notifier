package internal

import (
	_ "embed"
	"encoding/json"
)

//go:embed rules.json
var rulesJSON []byte

var AlertRules []Rule

func init() {
	if err := json.Unmarshal(rulesJSON, &AlertRules); err != nil {
		panic(err)
	}
}
