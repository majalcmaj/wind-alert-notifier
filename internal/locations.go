package internal

type Location struct {
	ID   string  `json:"id"   dynamodbav:"id"`
	Name string  `json:"name" dynamodbav:"name"`
	Lat  float64 `json:"lat"  dynamodbav:"lat"`
	Lon  float64 `json:"lon"  dynamodbav:"lon"`
}
