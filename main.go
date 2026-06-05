package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/majalcmaj/wind-alert-go/internal"
)

func handler(ctx context.Context, event events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	openWeatherToken := os.Getenv("OPENWEATHER_TOKEN")

	if len(strings.TrimSpace(openWeatherToken)) == 0 {
		return nil, errors.New("OPENWEATHER_TOKEN env variable must be set")
	}

	openWeather, err := internal.NewOpenWeather("https://api.openweathermap.org", openWeatherToken)
	if err != nil {
		return nil, err
	}
	forecast, err := openWeather.GetForecast(54.646034, 18.512407)

	if err != nil {
		return nil, err
	}

	triggered := internal.EvaluateForecast(forecast, internal.AlertRules)
	if len(triggered) == 0 {
		return &events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       "No rule matched — no mail sent",
		}, nil
	}

	descs := make([]string, len(triggered))
	for i, r := range triggered {
		descs[i] = r.Describe()
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	sesClient := sesv2.NewFromConfig(cfg)

	forecastJson, _ := json.MarshalIndent(forecast, "", "  ")
	mail, err := internal.RenderMail(internal.MailModel{Reading: forecast, TriggeredRules: descs})
	if err != nil {
		return nil, err
	}

	emailOutput, err := sesClient.SendEmail(ctx, &sesv2.SendEmailInput{
		Content: &types.EmailContent{
			Simple: &types.Message{
				Body: &types.Body{
					Html: &types.Content{
						Data:    aws.String(mail),
						Charset: aws.String("UTF-8"),
					},
				},
				Subject: &types.Content{
					Data: aws.String("Wind Forecast Alert"),
				},
			},
		},
		Destination: &types.Destination{
			ToAddresses: []string{"majalcmaj@gmail.com"},
		},
		FromEmailAddress: aws.String("m.w.ciesiel@gmail.com"),
	})

	if err != nil {
		return nil, err
	}

	emailOutputJson, _ := json.MarshalIndent(emailOutput, "", "  ")

	response := events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       fmt.Sprintf("%s\nEmail output: %s", forecastJson, emailOutputJson),
	}
	return &response, nil
}

func main() {
	lambda.Start(handler)
}
