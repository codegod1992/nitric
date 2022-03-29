// Copyright 2021 Nitric Pty Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package secrets_manager_secret_service

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	secretsmanager "github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"

	"github.com/nitrictech/nitric/pkg/plugins/errors"
	"github.com/nitrictech/nitric/pkg/plugins/errors/codes"
	"github.com/nitrictech/nitric/pkg/plugins/secret"
	"github.com/nitrictech/nitric/pkg/providers/aws/core"
	"github.com/nitrictech/nitric/pkg/utils"
)

type secretsManagerSecretService struct {
	secret.UnimplementedSecretPlugin
	client   secretsmanageriface.SecretsManagerAPI
	provider core.AwsProvider
}

func (s *secretsManagerSecretService) validateNewSecret(sec *secret.Secret, val []byte) error {
	if sec == nil {
		return fmt.Errorf("provide non-empty secret")
	}
	if len(sec.Name) == 0 {
		return fmt.Errorf("provide non-empty secret name")
	}
	if len(val) == 0 {
		return fmt.Errorf("provide non-empty secret value")
	}

	return nil
}

func (s *secretsManagerSecretService) getSecretId(sec string) (string, error) {
	secrets, err := s.provider.GetResources(core.AwsResource_Secret)

	if err != nil {
		return "", fmt.Errorf("error retrieving secrets list: %v", err)
	}

	if secret, ok := secrets[sec]; ok {
		return secret, nil
	}

	return "", fmt.Errorf("secret %s does not exist", sec)
}

func (s *secretsManagerSecretService) Put(sec *secret.Secret, val []byte) (*secret.SecretPutResponse, error) {
	newErr := errors.ErrorsWithScope(
		"SecretManagerSecretService.Put",
		map[string]interface{}{
			"secret": sec,
		},
	)

	if err := s.validateNewSecret(sec, val); err != nil {
		return nil, newErr(
			codes.InvalidArgument,
			"invalid secret",
			err,
		)
	}

	secretId, err := s.getSecretId(sec.Name)

	if err != nil {
		return nil, newErr(codes.NotFound, "unable to find secret", err)
	}

	result, err := s.client.PutSecretValue(&secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(secretId),
		SecretBinary: val,
	})

	if err != nil {
		return nil, newErr(codes.Internal, "unable to put secret", err)
	}

	return &secret.SecretPutResponse{
		SecretVersion: &secret.SecretVersion{
			Secret: &secret.Secret{
				Name: sec.Name,
			},
			Version: aws.StringValue(result.VersionId),
		},
	}, nil
}

func (s *secretsManagerSecretService) Access(sv *secret.SecretVersion) (*secret.SecretAccessResponse, error) {
	newErr := errors.ErrorsWithScope(
		"SecretManagerSecretService.Access",
		map[string]interface{}{
			"version": sv,
		},
	)

	if len(sv.Secret.Name) == 0 {
		return nil, newErr(
			codes.InvalidArgument,
			"provide non-empty secret name",
			nil,
		)
	}

	if len(sv.Version) == 0 {
		return nil, newErr(
			codes.InvalidArgument,
			"provide non-empty version",
			nil,
		)
	}

	secretId, err := s.getSecretId(sv.Secret.Name)

	if err != nil {
		return nil, newErr(codes.NotFound, "could not find secret", err)
	}

	//Build the request to get the secret
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretId),
	}

	// If the requested version is latest then we want
	// to exclude the version from input
	if strings.ToLower(sv.Version) != "latest" {
		input.VersionId = aws.String(sv.Version)
	}

	result, err := s.client.GetSecretValue(input)

	if err != nil {
		return nil, newErr(
			codes.NotFound,
			"failed to retrieve secret version",
			err,
		)
	}

	return &secret.SecretAccessResponse{
		SecretVersion: &secret.SecretVersion{
			Secret: &secret.Secret{
				Name: sv.Secret.Name,
			},
			Version: aws.StringValue(result.VersionId),
		},
		Value: result.SecretBinary,
	}, nil
}

//Gets a new Secrets Manager Client
func New(provider core.AwsProvider) (secret.SecretService, error) {
	awsRegion := utils.GetEnv("AWS_REGION", "us-east-1")

	sess, sessionError := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	})

	if sessionError != nil {
		return nil, fmt.Errorf("error creating new AWS session %v", sessionError)
	}

	client := secretsmanager.New(sess)

	return &secretsManagerSecretService{
		client:   client,
		provider: provider,
	}, nil
}
