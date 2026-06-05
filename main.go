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

type mailSender interface {
	send(ctx context.Context, subject, htmlBody string) error
}

type sesSender struct{ client *sesv2.Client }

func (s *sesSender) send(ctx context.Context, subject, htmlBody string) error {
	_, err := s.client.SendEmail(ctx, &sesv2.SendEmailInput{
		Content: &types.EmailContent{
			Simple: &types.Message{
				Body: &types.Body{
					Html: &types.Content{Data: aws.String(htmlBody), Charset: aws.String("UTF-8")},
				},
				Subject: &types.Content{Data: aws.String(subject)},
			},
		},
		Destination:      &types.Destination{ToAddresses: []string{"majalcmaj@gmail.com"}},
		FromEmailAddress: aws.String("m.w.ciesiel@gmail.com"),
	})
	return err
}

type stdoutSender struct{}

func (s *stdoutSender) send(_ context.Context, subject, htmlBody string) error {
	fmt.Printf("=== MAIL (local mode): %s ===\n%s\n=== END MAIL ===\n", subject, htmlBody)
	return nil
}

func newMailSender(ctx context.Context) (mailSender, error) {
	if os.Getenv("LOCAL_MODE") == "true" {
		return &stdoutSender{}, nil
	}
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &sesSender{client: sesv2.NewFromConfig(cfg)}, nil
}

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
		return &events.APIGatewayProxyResponse{StatusCode: 200, Body: "No rule matched — no mail sent"}, nil
	}

	descs := make([]string, len(triggered))
	for i, r := range triggered {
		descs[i] = r.Describe()
	}

	mail, err := internal.RenderMail(internal.MailModel{Reading: forecast, TriggeredRules: descs})
	if err != nil {
		return nil, err
	}

	sender, err := newMailSender(ctx)
	if err != nil {
		return nil, err
	}
	if err := sender.send(ctx, "Wind Forecast Alert", mail); err != nil {
		return nil, err
	}

	forecastJson, _ := json.MarshalIndent(forecast, "", "  ")
	return &events.APIGatewayProxyResponse{StatusCode: 200, Body: string(forecastJson)}, nil
}

func main() {
	lambda.Start(handler)
}
