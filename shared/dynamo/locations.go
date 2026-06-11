package dynamo

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/majalcmaj/wind-alert/shared/model"
)

func (s *Store) LoadLocations(ctx context.Context) ([]model.Location, error) {
	out, err := s.db.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(tableLocations),
	})
	if err != nil {
		return nil, err
	}
	var locs []model.Location
	return locs, attributevalue.UnmarshalListOfMaps(out.Items, &locs)
}

func (s *Store) PutLocation(ctx context.Context, loc model.Location) error {
	item, err := attributevalue.MarshalMap(loc)
	if err != nil {
		return err
	}
	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableLocations),
		Item:      item,
	})
	return err
}

func (s *Store) DeleteLocation(ctx context.Context, id string) error {
	_, err := s.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(tableLocations),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	return err
}
