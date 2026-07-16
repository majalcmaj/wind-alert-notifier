package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"wind-alert/internal/dynamo"
	"wind-alert/web/internal/server"
)

func main() {
	ctx := context.Background()
	st, err := dynamo.New(ctx)
	if err != nil {
		log.Fatal(err)
	}
	srv := server.New(st)
	handler := server.BasicAuth(srv.Routes())
	lambda.Start(httpadapter.NewV2(handler).ProxyWithContext)
}
