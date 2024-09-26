package config_aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
)

func GetAWSConfig(ctx context.Context, awsRegion string) (*aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(awsRegion))
	if err != nil {
		return nil, err
	}
	otelaws.AppendMiddlewares(&cfg.APIOptions)

	return &cfg, nil
}