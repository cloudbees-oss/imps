// Copyright © 2021 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ecr

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type StringableCredentials struct {
	aws.Credentials
	// Region specifies which region to connect to when using this credential
	Region string
	// Assume a role
	RoleArn string
}

func (c *StringableCredentials) GetCreds(_ context.Context) (aws.Credentials, error) {
	return c.Credentials, nil
}

func (c *StringableCredentials) ToAwsConfig() aws.Config {
	ctx := context.Background()

	// Always load default config to get HTTPClient and middleware
	// This is required for request signing to work properly
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(c.Region))
	if err != nil {
		// If LoadDefaultConfig fails completely, we can't proceed safely
		// Return a basic config and let the error surface later
		return aws.Config{Region: c.Region}
	}

	// If explicit credentials are provided, override the credential provider
	if c.AccessKeyID != "" {
		cfg.Credentials = aws.CredentialsProviderFunc(func(_ context.Context) (aws.Credentials, error) {
			return c.Credentials, nil
		})
	}
	// Otherwise, cfg already has credentials from default chain (including IRSA)

	// If roleArn is specified in the secret, assume that role
	if len(c.RoleArn) != 0 {
		// The cfg has either explicit credentials or IRSA credentials
		// Now assume the target role using those base credentials
		stsSvc := sts.NewFromConfig(cfg)
		creds := stscreds.NewAssumeRoleProvider(stsSvc, c.RoleArn)
		cfg.Credentials = aws.NewCredentialsCache(creds)
	}

	return cfg
}

func (c *StringableCredentials) Retrieve(_ context.Context) (aws.Credentials, error) {
	return c.Credentials, nil
}

func (c *StringableCredentials) String() string {
	return fmt.Sprintf("%s/%s/%s/%s", c.Region, c.AccessKeyID, c.SecretAccessKey, c.SessionToken)
}
