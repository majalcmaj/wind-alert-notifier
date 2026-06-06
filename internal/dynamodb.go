package internal

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	ddbexpr "github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type dynamoDBClient interface {
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

func LoadLocations(ctx context.Context, client dynamoDBClient) ([]Location, error) {
	out, err := client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String("wind-alert-locations"),
	})
	if err != nil {
		return nil, err
	}
	var locs []Location
	return locs, attributevalue.UnmarshalListOfMaps(out.Items, &locs)
}

func LoadRulesForLocation(ctx context.Context, client dynamoDBClient, locationID string) ([]Rule, error) {
	expr, err := ddbexpr.NewBuilder().
		WithKeyCondition(ddbexpr.Key("location_id").Equal(ddbexpr.Value(locationID))).
		Build()
	if err != nil {
		return nil, err
	}
	out, err := client.Query(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String("wind-alert-rules"),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return nil, err
	}
	var rules []Rule
	return rules, attributevalue.UnmarshalListOfMaps(out.Items, &rules)
}
