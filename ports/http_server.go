package ports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/flashbots/builder-hub/domain"
	"github.com/go-chi/httplog/v2"
)

type BuilderHubService interface {
	GetAllowedMeasurements(ctx context.Context) ([]domain.Measurement, error)
	GetActiveBuilders(ctx context.Context) ([]domain.BuilderWithServices, error)
	VerifyIpAndMeasurements(ctx context.Context, ip net.IP, measurement *domain.Measurement) (*domain.Builder, error)
}
type BuilderHubHandler struct {
	builderHubService BuilderHubService
	log               *httplog.Logger
}

func NewBuilderHubHandler(builderHubService BuilderHubService, log *httplog.Logger) *BuilderHubHandler {
	return &BuilderHubHandler{builderHubService: builderHubService, log: log}
}

type AuthData struct {
	AttestationType string
	MeasurementData map[string]string
	IP              net.IP
}

func (bhs *BuilderHubHandler) getAuthData(r *http.Request) (*AuthData, error) {
	attestationType := r.Header.Get(AttestationTypeHeader)
	if attestationType == "" {
		return nil, fmt.Errorf("attestation type is empty %w", ErrInvalidAuthData)
	}
	measurementHeader := r.Header.Get(MeasurementHeader)
	measurementData := make(map[string]string)
	err := json.Unmarshal([]byte(measurementHeader), &measurementData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal measurement header %w", ErrInvalidAuthData)
	}
	ipHeader := r.Header.Get(ForwardedHeader)
	ip := net.ParseIP(ipHeader)
	if ip == nil {
		return nil, fmt.Errorf("failed to parse ip %w", ErrInvalidAuthData)
	}

	return &AuthData{
		AttestationType: attestationType,
		MeasurementData: measurementData,
		IP:              ip,
	}, nil
}
func (bhs *BuilderHubHandler) GetAllowedMeasurements(w http.ResponseWriter, r *http.Request) {
	_, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	measurements, err := bhs.builderHubService.GetAllowedMeasurements(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var pMeasurements []Measurement
	for _, m := range measurements {
		pMeasurements = append(pMeasurements, fromDomainMeasurement(&m))
	}

	btsM, err := json.Marshal(measurements)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write(btsM)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (bhs *BuilderHubHandler) GetActiveBuilders(w http.ResponseWriter, r *http.Request) {
	//r.Header.Get("Authorization")
	authData, err := bhs.getAuthData(r)
	if err != nil {
		bhs.log.Warn("malformed auth data", "error", err)
		w.WriteHeader(http.StatusForbidden)
		return
	}
	_, err = bhs.builderHubService.VerifyIpAndMeasurements(r.Context(), authData.IP, domain.NewMeasurement(authData.AttestationType, authData.MeasurementData))
	if errors.Is(err, domain.ErrNotFound) {
		bhs.log.Warn("invalid auth data", "error", err)
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if err != nil {
		bhs.log.Error("failed to verify ip and measurements", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	builders, err := bhs.builderHubService.GetActiveBuilders(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var pBuilders []BuilderWithServiceCreds
	for _, b := range builders {
		pBuilders = append(pBuilders, fromDomainBuilderWithServices(&b))
	}
	bts, err := json.Marshal(pBuilders)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write(bts)
}

func (bhs *BuilderHubHandler) GetActiveBuildersNoAuth(w http.ResponseWriter, r *http.Request) {
	//bhs.builderHubService
	builders, err := bhs.builderHubService.GetActiveBuilders(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var pBuilders []BuilderWithServiceCreds
	for _, b := range builders {
		pBuilders = append(pBuilders, fromDomainBuilderWithServices(&b))
	}
	bts, err := json.Marshal(pBuilders)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write(bts)
}

func (bhs *BuilderHubHandler) GetConfigSecrets(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (bhs *BuilderHubHandler) RegisterCredentials(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
