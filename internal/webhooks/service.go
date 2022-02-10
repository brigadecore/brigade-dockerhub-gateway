package webhooks

import (
	"context"
	"encoding/json"

	"github.com/brigadecore/brigade/sdk/v3"
	"github.com/go-playground/webhooks/v6/docker"
	"github.com/pkg/errors"
)

// Service is an interface for components that can handle webhooks (events) from
// Docker Hub. Implementations of this interface are transport-agnostic.
type Service interface {
	// Handle handles a webhook (event) from Docker Hub.
	Handle(context.Context, docker.BuildPayload) error
}

type service struct {
	eventsClient sdk.EventsClient
}

// NewService returns an implementation of the Service interface for handling
// webhooks (events) from Docker Hub.
func NewService(eventsClient sdk.EventsClient) Service {
	return &service{
		eventsClient: eventsClient,
	}
}

func (s *service) Handle(
	ctx context.Context,
	payload docker.BuildPayload,
) error {
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "error marshaling payload")
	}
	event := sdk.Event{
		Source: "brigade.sh/dockerhub",
		Type:   "push",
		Qualifiers: map[string]string{
			"repo": payload.Repository.RepoName,
		},
		Payload: string(rawPayload),
	}
	_, err = s.eventsClient.Create(context.Background(), event, nil)
	return errors.Wrap(err, "error emitting event(s) into Brigade")
}
