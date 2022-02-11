package webhooks

import (
	"context"
	"testing"

	"github.com/brigadecore/brigade/sdk/v3"
	sdkTesting "github.com/brigadecore/brigade/sdk/v3/testing"
	"github.com/go-playground/webhooks/v6/docker"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	s, ok := NewService(
		// Totally unusable client that is enough to fulfill the dependencies for
		// this test...
		&sdkTesting.MockEventsClient{
			LogsClient: &sdkTesting.MockLogsClient{},
		},
	).(*service)
	require.True(t, ok)
	require.NotNil(t, s.eventsClient)
}

func TestHandle(t *testing.T) {
	const testRepoName = "brigadecore/brigade-dockerhub-gateway"
	testBuildPayload := docker.BuildPayload{}
	testBuildPayload.Repository.RepoName = testRepoName
	testCases := []struct {
		name       string
		service    *service
		assertions func(error)
	}{
		{
			name: "error creating brigade event",
			service: &service{
				eventsClient: &sdkTesting.MockEventsClient{
					CreateFn: func(
						context.Context,
						sdk.Event,
						*sdk.EventCreateOptions,
					) (sdk.EventList, error) {
						return sdk.EventList{}, errors.New("something went wrong")
					},
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error emitting event(s) into Brigade",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "success",
			service: &service{
				eventsClient: &sdkTesting.MockEventsClient{
					CreateFn: func(
						_ context.Context,
						event sdk.Event,
						_ *sdk.EventCreateOptions,
					) (sdk.EventList, error) {
						require.Equal(t, "brigade.sh/dockerhub", event.Source)
						require.Equal(t, "push", event.Type)
						require.Equal(
							t,
							map[string]string{
								"repo": testRepoName,
							},
							event.Qualifiers,
						)
						require.NotEmpty(t, event.Payload)
						return sdk.EventList{}, nil
					},
				},
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.service.Handle(context.Background(), testBuildPayload)
			testCase.assertions(err)
		})
	}
}
