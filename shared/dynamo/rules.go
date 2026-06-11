package dynamo

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	ddbexpr "github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/majalcmaj/wind-alert/shared/model"
)

func (s *Store) LoadRulesForLocation(ctx context.Context, locationID string) ([]model.Rule, error) {
	expr, err := ddbexpr.NewBuilder().
		WithKeyCondition(ddbexpr.Key("location_id").Equal(ddbexpr.Value(locationID))).
		Build()
	if err != nil {
		return nil, err
	}
	out, err := s.db.Query(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String(tableRules),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return nil, err
	}
	var rules []model.Rule
	return rules, attributevalue.UnmarshalListOfMaps(out.Items, &rules)
}

func (s *Store) PutRule(ctx context.Context, rule model.Rule) error {
	item, err := attributevalue.MarshalMap(rule)
	if err != nil {
		return err
	}
	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableRules),
		Item:      item,
	})
	return err
}

func (s *Store) DeleteRule(ctx context.Context, locationID, name string) error {
	_, err := s.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(tableRules),
		Key: map[string]types.AttributeValue{
			"location_id": &types.AttributeValueMemberS{Value: locationID},
			"name":        &types.AttributeValueMemberS{Value: name},
		},
	})
	return err
}

func (s *Store) DeleteRulesForLocation(ctx context.Context, locationID string) error {
	rules, err := s.LoadRulesForLocation(ctx, locationID)
	if err != nil {
		return err
	}
	if len(rules) == 0 {
		return nil
	}

	const batchSize = 25
	for i := 0; i < len(rules); i += batchSize {
		end := i + batchSize
		if end > len(rules) {
			end = len(rules)
		}
		chunk := rules[i:end]

		reqs := make([]types.WriteRequest, len(chunk))
		for j, r := range chunk {
			reqs[j] = types.WriteRequest{
				DeleteRequest: &types.DeleteRequest{
					Key: map[string]types.AttributeValue{
						"location_id": &types.AttributeValueMemberS{Value: r.LocationID},
						"name":        &types.AttributeValueMemberS{Value: r.Name},
					},
				},
			}
		}
		_, err = s.db.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				tableRules: reqs,
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}
