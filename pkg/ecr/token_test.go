package ecr

import (
	"context"
	"testing"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/stretchr/testify/mock"
	"gotest.tools/assert"
)

type MockECRClient struct {
	mock.Mock
}

func (m *MockECRClient) GetAuthorizationToken(ctx context.Context, input *ecr.GetAuthorizationTokenInput, _ ...func(*ecr.Options)) (*ecr.GetAuthorizationTokenOutput, error) {
	args := m.Called(ctx, input)
	// nolint:forcetypeassert
	return args.Get(0).(*ecr.GetAuthorizationTokenOutput), args.Error(1)
}

func TestToken_NewECRToken(t *testing.T) {
	t.Parallel()
	type args struct {
		creds  StringableCredentials
		client ClientInterface
	}

	mockClient := &MockECRClient{}
	testTokenName := "testToken"
	mockTokenOutput := &ecr.GetAuthorizationTokenOutput{
		AuthorizationData: []types.AuthorizationData{
			{
				AuthorizationToken: &testTokenName,
			},
		},
	}
	mockClient.On("GetAuthorizationToken", mock.Anything, mock.Anything).Return(mockTokenOutput, nil)

	mockClientEmptyData := &MockECRClient{}
	mockClientEmptyData.On("GetAuthorizationToken", mock.Anything, mock.Anything).Return(&ecr.GetAuthorizationTokenOutput{
		AuthorizationData: nil,
	}, nil)

	tests := []struct {
		name        string
		args        args
		want        *Token
		expectedErr error
	}{
		{
			name: "basic functionality test",
			args: args{
				creds:  StringableCredentials{},
				client: mockClient,
			},
			want: &Token{
				CurrentToken: &types.AuthorizationData{
					AuthorizationToken: &testTokenName,
				},
			},
			expectedErr: nil,
		},
		{
			name: "no token returned",
			args: args{
				creds:  StringableCredentials{},
				client: mockClientEmptyData,
			},
			want:        &Token{},
			expectedErr: errors.New("no authorization data is returned from ECR"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			found, err := NewECRToken(t.Context(), tt.args.creds, tt.args.client)

			if tt.expectedErr != nil {
				assert.Assert(t, err != nil, "expected an error")
				assert.Assert(t, found == nil, "expected token to be nil on error")
			} else {
				assert.NilError(t, err)
				// Compare only exported fields to avoid issues with unexported fields in newer SDK
				assert.Equal(t, tt.want.CurrentToken.AuthorizationToken, found.CurrentToken.AuthorizationToken)
				assert.Equal(t, tt.want.CurrentToken.ProxyEndpoint, found.CurrentToken.ProxyEndpoint)
				assert.Equal(t, tt.want.CurrentToken.ExpiresAt, found.CurrentToken.ExpiresAt)
			}
		})
	}
}

func TestToken_Refresh(t *testing.T) {
	t.Parallel()

	testTokenName := "testToken"

	tests := []struct {
		name            string
		mockTokenOutput *ecr.GetAuthorizationTokenOutput
		token           *Token
		expectedErr     error
	}{
		{
			name: "basic functionality test",
			mockTokenOutput: &ecr.GetAuthorizationTokenOutput{
				AuthorizationData: []types.AuthorizationData{
					{
						AuthorizationToken: &testTokenName,
					},
				},
			},
			token:       &Token{},
			expectedErr: nil,
		},
		{
			name: "no authorization data",
			mockTokenOutput: &ecr.GetAuthorizationTokenOutput{
				AuthorizationData: nil,
			},
			token:       &Token{},
			expectedErr: errors.New("no authorization data is returned from ECR"),
		},
		{
			name: "multiple authorization records",
			mockTokenOutput: &ecr.GetAuthorizationTokenOutput{
				AuthorizationData: []types.AuthorizationData{
					{
						AuthorizationToken: &testTokenName,
					},
					{
						AuthorizationToken: &testTokenName,
					},
				},
			},
			token:       &Token{},
			expectedErr: errors.New("multiple authorization records are returned for ECR"),
		},
		{
			name: "authorization token is empty",
			mockTokenOutput: &ecr.GetAuthorizationTokenOutput{
				AuthorizationData: []types.AuthorizationData{{}},
			},
			token:       &Token{},
			expectedErr: errors.New("no authorization data is returned from ECR - authorization token is empty"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockClient := &MockECRClient{}
			mockClient.On("GetAuthorizationToken", mock.Anything, mock.Anything).Return(tt.mockTokenOutput, nil)
			tt.token.Client = mockClient

			err := tt.token.Refresh(t.Context())

			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NilError(t, err)
			}
		})
	}
}
