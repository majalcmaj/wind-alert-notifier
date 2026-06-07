package internal

import (
	"bytes"
	_ "embed"
	"html/template"
)

//go:embed mail_template.html
var mailTemplate string

var tpl, parseErr = template.New("mail").Funcs(template.FuncMap{"windArrow": renderWindArrow}).Parse(mailTemplate)

func init() {
	if parseErr != nil {
		panic(parseErr)
	}
}

type ProviderIssue struct {
	Name  string
	Error string
}

type LocationResult struct {
	Location       Location
	Reading        *WeatherReading
	TriggeredRules []ConfidentRule
	ProviderIssues []ProviderIssue
}

type MailModel struct {
	Results []LocationResult
}

func RenderMail(model MailModel) (string, error) {
	buffer := new(bytes.Buffer)
	err := tpl.Execute(buffer, model)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}
