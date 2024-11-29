package secret_manager_aws

import (
	"context"
	
	"github.com/rs/zerolog/log"

	"github.com/go-fund-transfer/internal/lib"
	"github.com/go-fund-transfer/internal/core"
	"github.com/go-fund-transfer/internal/config/config_aws"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

var childLogger = log.With().Str("repository", "AwsClientSecretManager").Logger()

type AwsClientSecretManager struct {
	Client *secretsmanager.Client
}

func NewClientSecretManager(ctx context.Context, awsServiceConfig core.AwsServiceConfig) (*AwsClientSecretManager, error) {
	childLogger.Debug().Msg("NewClientSecretManager")

	span := lib.Span(ctx, "repository.NewClientSecretManager")	
    defer span.End()
	
	sdkConfig, err := config_aws.GetAWSConfig(ctx, awsServiceConfig.AwsRegion)
	if err != nil{
		return nil, err
	}
	
	client := secretsmanager.NewFromConfig(*sdkConfig)
	return &AwsClientSecretManager{
		Client: client,
	}, nil
}

func (p *AwsClientSecretManager) GetSecret(ctx context.Context, secretName string) (*string, error) {
	result, err := p.Client.GetSecretValue(ctx, 
										&secretsmanager.GetSecretValueInput{
											SecretId:		aws.String(secretName),
											VersionStage:	aws.String("AWSCURRENT"),
										})
	if err != nil {
		return nil, err
	}
	return result.SecretString, nil
}