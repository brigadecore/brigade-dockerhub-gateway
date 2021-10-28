package webhooks

import (
	"net/http"

	"github.com/go-playground/webhooks/v6/docker"
	"github.com/pkg/errors"
)

// handler is an implementation of the http.Handler interface that can handle
// webhooks (events) from Docker Hub by delegating to a transport-agnostic
// Service interface.
type handler struct {
	service Service
	hook    *docker.Webhook
}

// handler is an implementation of the http.Handler interface that can handle
// webhooks (events) from Docker Hub by delegating to a transport-agnostic
// Service interface.
func NewHandler(service Service) (http.Handler, error) {
	hook, err := docker.New()
	if err != nil {
		return nil, errors.Wrap(err, "error creating handler")
	}
	return &handler{
		service: service,
		hook:    hook,
	}, nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	payload, err := h.hook.Parse(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		// nolint: errcheck
		w.Write([]byte(`{"status": "error parsing request body"}`))
		return
	}
	// This will always be a docker.BuildPayload:
	dockerPayload := payload.(docker.BuildPayload) // nolint: forcetypeassert
	if err := h.service.Handle(r.Context(), dockerPayload); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status": "internal server error"}`)) // nolint: errcheck
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "success"}`)) // nolint: errcheck
}
