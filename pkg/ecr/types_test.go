package ecr

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"gotest.tools/assert"
)

func TestStringableCredentials_GetCreds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                  string
		stringableCredentials *StringableCredentials
		want                  aws.Credentials
	}{
		{
			name:                  "empty credentials",
			stringableCredentials: &StringableCredentials{},
			want:                  aws.Credentials{},
		},
		{
			name: "non-empty credentials",
			stringableCredentials: &StringableCredentials{
				aws.Credentials{
					AccessKeyID: "testAccessKeyID",
				},
				"testRegion",
				"testRole",
			},
			want: aws.Credentials{
				AccessKeyID: "testAccessKeyID",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			found, err := tt.stringableCredentials.GetCreds(t.Context())

			assert.Equal(t, tt.want, found)
			assert.NilError(t, err)
		})
	}
}

func TestStringableCredentials_ToAwsConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                  string
		stringableCredentials *StringableCredentials
		wantRegion            string
		wantExplicitCreds     bool
		wantAssumeRole        bool
	}{
		{
			name: "IRSA mode - empty credentials, no role",
			stringableCredentials: &StringableCredentials{
				Region: "testRegion",
			},
			wantRegion:        "testRegion",
			wantExplicitCreds: false,
			wantAssumeRole:    false,
		},
		{
			name: "IRSA with AssumeRole - empty credentials, with role",
			stringableCredentials: &StringableCredentials{
				Region:  "testRegion",
				RoleArn: "arn:aws:iam::123456789012:role/test-role",
			},
			wantRegion:        "testRegion",
			wantExplicitCreds: false,
			wantAssumeRole:    true,
		},
		{
			name: "explicit credentials with role",
			stringableCredentials: &StringableCredentials{
				Credentials: aws.Credentials{
					AccessKeyID:     "ASIA_TEST_KEY_EXAMPLE",
					SecretAccessKey: "test_secret_key_for_unit_tests_only",
				},
				Region:  "testRegion",
				RoleArn: "arn:aws:iam::123456789012:role/test-role",
			},
			wantRegion:        "testRegion",
			wantExplicitCreds: true,
			wantAssumeRole:    true,
		},
		{
			name: "explicit credentials without role",
			stringableCredentials: &StringableCredentials{
				Credentials: aws.Credentials{
					AccessKeyID:     "ASIA_TEST_KEY_EXAMPLE",
					SecretAccessKey: "test_secret_key_for_unit_tests_only",
				},
				Region: "testRegion",
			},
			wantRegion:        "testRegion",
			wantExplicitCreds: true,
			wantAssumeRole:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			found := tt.stringableCredentials.ToAwsConfig()

			assert.Equal(t, tt.wantRegion, found.Region)

			// Check if credentials are set
			// Credentials should always be set (either explicit or from default chain)
			assert.Assert(t, found.Credentials != nil, "expected credentials to be set")
		})
	}
}

func TestStringableCredentials_Retrieve(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                  string
		stringableCredentials *StringableCredentials
		want                  aws.Credentials
	}{
		{
			name:                  "empty credentials",
			stringableCredentials: &StringableCredentials{},
			want:                  aws.Credentials{},
		},
		{
			name: "non-empty credentials",
			stringableCredentials: &StringableCredentials{
				aws.Credentials{
					AccessKeyID: "testAccessKeyID",
				},
				"testRegion",
				"testRole",
			},
			want: aws.Credentials{
				AccessKeyID: "testAccessKeyID",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			found, err := tt.stringableCredentials.Retrieve(t.Context())

			assert.Equal(t, tt.want, found)
			assert.NilError(t, err)
		})
	}
}

func TestStringableCredentials_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                  string
		stringableCredentials *StringableCredentials
		want                  string
	}{
		{
			name:                  "empty credentials",
			stringableCredentials: &StringableCredentials{},
			want:                  "///",
		},
		{
			name: "partially empty credentials",
			stringableCredentials: &StringableCredentials{
				aws.Credentials{
					AccessKeyID: "testAccessKeyID",
				},
				"testRegion",
				"testRole",
			},
			want: "testRegion/testAccessKeyID//",
		},
		{
			name: "non-empty credentials",
			stringableCredentials: &StringableCredentials{
				aws.Credentials{
					AccessKeyID:     "testAccessKeyID",
					SecretAccessKey: "testSecretAccessKey",
					SessionToken:    "testSessionToken",
				},
				"testRegion",
				"testRole",
			},
			want: "testRegion/testAccessKeyID/testSecretAccessKey/testSessionToken",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			found := tt.stringableCredentials.String()

			assert.Equal(t, tt.want, found)
		})
	}
}
