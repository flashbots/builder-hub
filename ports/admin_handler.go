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
	SetSecretValues(builderName string, values map[string]any) error
	GetSecretValues(builderName string) (map[string]string, error)
}

type AdminHandler struct {
	builderService AdminBuilderService
	secretService  AdminSecretService
	log            *httplog.Logger
}

func NewAdminHandler(service AdminBuilderService, secretService AdminSecretService, log *httplog.Logger) *AdminHandler {
	return &AdminHandler{builderService: service, secretService: secretService, log: log}
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
	configBts, err := s.builderService.GetActiveConfigForBuilder(r.Context(), builderName)
	if err != nil {
		s.log.Error("failed to get config with secrets", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	secrets, err := s.secretService.GetSecretValues(builderName)
	if err != nil {
		s.log.Error("failed to get secrets", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	bts, err := application.MergeConfigSecrets(configBts, secrets)
	if err != nil {
		s.log.Error("failed to merge config with secrets", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write(bts)
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
		s.log.Error("Failed to unmarshal request body", "err", err)
		w.WriteHeader(http.StatusBadRequest)
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
		s.log.Error("Failed to unmarshal request body", "err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	dBuilder, err := toDomainBuilder(builder, false)
	if err != nil {
		s.log.Error("Failed to convert builder to domain builder", "err", err)
		w.WriteHeader(http.StatusBadRequest)
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
		s.log.Error("failed to decode request body", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
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
		s.log.Error("failed to decode request body", "error", err)
		w.WriteHeader(http.StatusBadRequest)
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
		s.log.Error("Invalid json", "err", err)
		w.WriteHeader(http.StatusBadRequest)
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

	secretValues, err := application.FlattenJSONFromBytes(body)
	if err != nil {
		s.log.Error("Failed to flatten JSON", "err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = s.secretService.SetSecretValues(builderName, secretValues)
	if err != nil {
		s.log.Error("failed to set secret", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
