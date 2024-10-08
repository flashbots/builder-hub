package ports

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/flashbots/builder-hub/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog/v2"
)

type BuilderHubService interface {
	GetAllowedMeasurements(ctx context.Context) ([]domain.Measurement, error)
	GetActiveBuilders(ctx context.Context) ([]domain.BuilderWithServices, error)
	VerifyIpAndMeasurements(ctx context.Context, ip net.IP, measurement *domain.Measurement) (*domain.Builder, error)
	GetConfigWithSecrets(ctx context.Context, builderName string) ([]byte, error)
	RegisterCredentialsForBuilder(ctx context.Context, builderName, service, tlsCert string, ecdsaPubKey, measurementHash []byte, attestationType string) error
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
	MeasurementData map[string]domain.SingleMeasurement
	IP              net.IP
}

func (bhs *BuilderHubHandler) getAuthData(r *http.Request) (*AuthData, error) {
	attestationType := r.Header.Get(AttestationTypeHeader)
	if attestationType == "" {
		return nil, fmt.Errorf("attestation type is empty %w", ErrInvalidAuthData)
	}
	measurementHeader := r.Header.Get(MeasurementHeader)
	measurementData := make(map[string]domain.SingleMeasurement)
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
		bhs.log.Error("failed to read request body", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	measurements, err := bhs.builderHubService.GetAllowedMeasurements(r.Context())
	if err != nil {
		bhs.log.Error("failed to fetch allowed measurements from db", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var pMeasurements []Measurement
	for _, m := range measurements {
		pMeasurements = append(pMeasurements, fromDomainMeasurement(&m))
	}

	btsM, err := json.Marshal(pMeasurements)
	if err != nil {
		bhs.log.Error("failed to marshal measurements", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write(btsM)
	if err != nil {
		bhs.log.Error("failed to write response", "error", err)
	}
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
		bhs.log.Error("failed to fetch active builders from db", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var pBuilders []BuilderWithServiceCreds
	for _, b := range builders {
		pBuilders = append(pBuilders, fromDomainBuilderWithServices(&b))
	}
	bts, err := json.Marshal(pBuilders)
	if err != nil {
		bhs.log.Error("failed to marshal builders", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write(bts)
	if err != nil {
		bhs.log.Error("failed to write response", "error", err)
	}
}

func (bhs *BuilderHubHandler) GetActiveBuildersNoAuth(w http.ResponseWriter, r *http.Request) {
	//bhs.builderHubService
	builders, err := bhs.builderHubService.GetActiveBuilders(r.Context())
	if err != nil {
		bhs.log.Error("failed to fetch active builders from db", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var pBuilders []BuilderWithServiceCreds
	for _, b := range builders {
		pBuilders = append(pBuilders, fromDomainBuilderWithServices(&b))
	}
	bts, err := json.Marshal(pBuilders)
	if err != nil {
		bhs.log.Error("failed to marshal builders", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write(bts)
	if err != nil {
		bhs.log.Error("failed to write response", "error", err)
	}
}

func (bhs *BuilderHubHandler) GetConfigSecrets(w http.ResponseWriter, r *http.Request) {
	authData, err := bhs.getAuthData(r)
	if err != nil {
		bhs.log.Warn("malformed auth data", "error", err)
		w.WriteHeader(http.StatusForbidden)
		return
	}
	builder, err := bhs.builderHubService.VerifyIpAndMeasurements(r.Context(), authData.IP, domain.NewMeasurement(authData.AttestationType, authData.MeasurementData))
	if errors.Is(err, domain.ErrNotFound) {
		bhs.log.Warn("invalid auth data", "error", err)
		w.WriteHeader(http.StatusForbidden)
		return
	}
	bts, err := bhs.builderHubService.GetConfigWithSecrets(r.Context(), builder.Name)
	if err != nil {
		bhs.log.Error("failed to get config with secrets", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write(bts)
	if err != nil {
		bhs.log.Error("failed to write response", "error", err)
	}
}

func (bhs *BuilderHubHandler) RegisterCredentials(w http.ResponseWriter, r *http.Request) {
	authData, err := bhs.getAuthData(r)
	if err != nil {
		bhs.log.Warn("malformed auth data", "error", err)
		w.WriteHeader(http.StatusForbidden)
		return
	}
	measurement := domain.NewMeasurement(authData.AttestationType, authData.MeasurementData)
	builder, err := bhs.builderHubService.VerifyIpAndMeasurements(r.Context(), authData.IP, measurement)
	if errors.Is(err, domain.ErrNotFound) {
		bhs.log.Warn("invalid auth data", "error", err)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	service := chi.URLParam(r, "service")
	//TODO: validate service
	if service == "" {
		bhs.log.Warn("service is empty")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		bhs.log.Error("Failed to read request body", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	sc := ServiceCred{}
	err = json.Unmarshal(body, &sc)
	if err != nil {
		bhs.log.Error("Failed to unmarshal request body", "err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	ecdsaBytes, err := hex.DecodeString(sc.EcdsaPubkey)
	if err != nil {
		bhs.log.Error("Failed to decode ecdsa public key", "err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = bhs.builderHubService.RegisterCredentialsForBuilder(r.Context(), builder.Name, service, sc.TlsCert, ecdsaBytes, measurement.Hash, measurement.AttestationType)
	if err != nil {
		bhs.log.Error("Failed to register credentials", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
