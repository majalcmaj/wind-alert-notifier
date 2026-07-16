module wind-alert

go 1.26

require (
	github.com/aws/aws-lambda-go v1.49.0
	github.com/awslabs/aws-lambda-go-api-proxy v0.16.2
)

replace github.com/majalcmaj/wind-alert/shared => ../shared
