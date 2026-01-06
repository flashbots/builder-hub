package ports

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/flashbots/builder-hub/application"
	"github.com/flashbots/builder-hub/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog/v2"
)

type AdminBuilderService interface {
	GetActiveConfigForBuilder(ctx context.Context, builderName string) (json.RawMessage, error)
	AddMeasurement(ctx context.Context, measurement domain.Measurement, enabled bool) error
	AddBuilder(ctx context.Context, builder domain.Builder) error
	ChangeActiveStatusForBuilder(ctx context.Context, builderName string, isActive bool) error
	ChangeActiveStatusForMeasurement(ctx context.Context, measurementName string, isActive bool) error
	AddBuilderConfig(ctx context.Context, builderName string, config json.RawMessage) error
}

type AdminSecretService interface {
	SetSecretValues(builderName string, message json.RawMessage) error
	application.SecretAccessor
}

type AdminHandler struct {
	builderService AdminBuilderService
	secretService  AdminSecretService
	handler
}

func NewAdminHandler(service AdminBuilderService, secretService AdminSecretService, log *httplog.Logger) *AdminHandler {
	return &AdminHandler{builderService: service, secretService: secretService, handler: handler{log: log}}
}

func (s *AdminHandler) GetActiveConfigForBuilder(w http.ResponseWriter, r *http.Request) {
	builderName := chi.URLParam(r, "builderName")
	bts, err := s.builderService.GetActiveConfigForBuilder(r.Context(), builderName)
	if errors.Is(err, domain.ErrNotFound) {
		s.log.Warn("no active config for builder found", "error", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		s.log.Error("failed to fetch active config for builder", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write(bts)
	if err != nil {
		s.log.Error("failed to write response", "error", err)
	}
}

// GetFullConfigForBuilder returns the full config for a builder, including secrets
// Note this copies logic from GetConfigWithSecrets in BuilderHubService
// since we decided to avoid application layer here it probably makes sense unless
// logic gets more complicated here
func (s *AdminHandler) GetFullConfigForBuilder(w http.ResponseWriter, r *http.Request) {
	builderName := chi.URLParam(r, "builderName")
	_, err := s.builderService.GetActiveConfigForBuilder(r.Context(), builderName)
	if err != nil {
		s.log.Error("failed to get config with secrets", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	secr, err := s.secretService.GetSecretValues(builderName)
	if err != nil {
		s.log.Error("failed to get secrets", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write(secr)
	if err != nil {
		s.log.Error("failed to write response", "error", err)
	}
}

func (s *AdminHandler) AddMeasurement(w http.ResponseWriter, r *http.Request) {
	// read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.log.Error("Failed to read request body", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	measurement := Measurement{}
	err = json.Unmarshal(body, &measurement)
	if err != nil {
		s.BadRequest(w, r, "failed to unmarshal request body", err)
		return
	}
	err = s.builderService.AddMeasurement(r.Context(), toDomainMeasurement(measurement), false)
	if err != nil {
		s.log.Error("failed to add measurement", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *AdminHandler) AddBuilder(w http.ResponseWriter, r *http.Request) {
	// read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.log.Error("Failed to read request body", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	builder := Builder{}
	err = json.Unmarshal(body, &builder)
	if err != nil {
		s.BadRequest(w, r, "failed to unmarshal request body", err)
		return
	}
	if builder.Network == "" {
		s.BadRequest(w, r, "network field is required")
		return
	}
	dBuilder, err := toDomainBuilder(builder, false)
	if err != nil {
		s.BadRequest(w, r, "failed to convert builder to domain builder", err)
		return
	}
	err = s.builderService.AddBuilder(r.Context(), dBuilder)
	if err != nil {
		s.log.Error("failed to add builder", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type ActivationRequest struct {
	Enabled bool `json:"enabled"`
}

func (s *AdminHandler) ChangeActiveStatusForBuilder(w http.ResponseWriter, r *http.Request) {
	builderName := chi.URLParam(r, "builderName")
	activationRequest := ActivationRequest{}
	err := json.NewDecoder(r.Body).Decode(&activationRequest)
	if err != nil {
		s.BadRequest(w, r, "failed to decode request body", err)
		return
	}

	// we only ensure existence of active config for `activation` request
	// `deactivation` request must pass through for an ease of deactivation incorrect/rouge deployments
	if activationRequest.Enabled {
		_, err = s.builderService.GetActiveConfigForBuilder(r.Context(), builderName)
		if errors.Is(err, domain.ErrNotFound) {
			s.BadRequest(w, r, "active config not found: please set config for the builder first")
			return
		}
		if err != nil {
			s.log.Error("failed to fetch active config for builder", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	err = s.builderService.ChangeActiveStatusForBuilder(r.Context(), builderName, activationRequest.Enabled)
	if err != nil {
		s.log.Error("failed to change active status for builder", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *AdminHandler) ChangeActiveStatusForMeasurement(w http.ResponseWriter, r *http.Request) {
	measurementName := chi.URLParam(r, "measurementName")
	activationRequest := ActivationRequest{}
	err := json.NewDecoder(r.Body).Decode(&activationRequest)
	if err != nil {
		s.BadRequest(w, r, "failed to decode request body", err)
		return
	}
	err = s.builderService.ChangeActiveStatusForMeasurement(r.Context(), measurementName, activationRequest.Enabled)
	if err != nil {
		s.log.Error("failed to change active status for measurement", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *AdminHandler) AddBuilderConfig(w http.ResponseWriter, r *http.Request) {
	builderName := chi.URLParam(r, "builderName")
	// read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.log.Error("Failed to read request body", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !json.Valid(body) {
		s.BadRequest(w, r, "invalid json")
		return
	}

	// validate valid json
	err = s.builderService.AddBuilderConfig(r.Context(), builderName, body)
	if err != nil {
		s.log.Error("failed to add builder config", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *AdminHandler) SetSecrets(w http.ResponseWriter, r *http.Request) {
	builderName := chi.URLParam(r, "builderName")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.log.Error("Failed to read request body", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = s.secretService.SetSecretValues(builderName, body)
	if err != nil {
		s.log.Error("failed to set secret", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
