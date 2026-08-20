package ports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/flashbots/builder-hub/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog/v2"
)

type BuilderHubService interface {
	GetAllowedMeasurements(ctx context.Context) ([]domain.Measurement, error)
	GetActiveBuilders(ctx context.Context, network string) ([]domain.BuilderWithServices, error)
	VerifyIPAndMeasurements(ctx context.Context, ip net.IP, measurement domain.SuppliedMeasurements, attestationType string) (*domain.Builder, string, error)
	GetConfigWithSecrets(ctx context.Context, builderName string) ([]byte, error)
	RegisterCredentialsForBuilder(ctx context.Context, builderName, service, tlsCert string, ecdsaPubKey []byte, measurementName, attestationType, region string) error
	LogEvent(ctx context.Context, eventName, builderName, name string) error
}
type BuilderHubHandler struct {
	builderHubService BuilderHubService
	handler
}

func NewBuilderHubHandler(builderHubService BuilderHubService, log *httplog.Logger) *BuilderHubHandler {
	return &BuilderHubHandler{builderHubService: builderHubService, handler: handler{log: log}}
}

type AuthData struct {
	AttestationType string
	MeasurementData domain.SuppliedMeasurements
	IP              net.IP
}

type expectedMeasurementsHeader struct {
	Type         domain.ExpectedMeasurementType `json:"type"`
	Measurements json.RawMessage                `json:"measurements"`
}

func parseMeasurementHeader(value, attestationType string) (domain.SuppliedMeasurements, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return domain.SuppliedMeasurements{}, err
	}
	if object == nil {
		return domain.SuppliedMeasurements{}, errors.New("measurement header must be a JSON object")
	}

	if _, typed := object["type"]; !typed {
		legacy := make(map[string]string)
		if err := json.Unmarshal([]byte(value), &legacy); err != nil {
			return domain.SuppliedMeasurements{}, err
		}
		return domain.SuppliedMeasurements{
			Type:            domain.ExpectedMeasurementLegacy,
			Registers:       nil,
			DcapImageHashes: nil,
			Legacy:          legacy,
		}, nil
	}

	var header expectedMeasurementsHeader
	if err := json.Unmarshal([]byte(value), &header); err != nil {
		return domain.SuppliedMeasurements{}, err
	}
	if !measurementTypeMatchesAttestationType(header.Type, attestationType) {
		return domain.SuppliedMeasurements{}, fmt.Errorf("measurement type %q does not match attestation type %q", header.Type, attestationType)
	}

	supplied := domain.SuppliedMeasurements{Type: header.Type}
	switch header.Type {
	case domain.ExpectedMeasurementDCAP, domain.ExpectedMeasurementAzure:
		if len(header.Measurements) == 0 || string(header.Measurements) == "null" {
			return domain.SuppliedMeasurements{}, errors.New("measurements are missing")
		}
		if err := json.Unmarshal(header.Measurements, &supplied.Registers); err != nil {
			return domain.SuppliedMeasurements{}, err
		}
		if supplied.Registers == nil {
			return domain.SuppliedMeasurements{}, errors.New("measurements must be an object")
		}
		for key, values := range supplied.Registers {
			if len(values) == 0 {
				return domain.SuppliedMeasurements{}, fmt.Errorf("measurement %q has no expected values", key)
			}
		}
	case domain.ExpectedMeasurementImage:
		if len(header.Measurements) == 0 || string(header.Measurements) == "null" {
			return domain.SuppliedMeasurements{}, errors.New("image measurements are missing")
		}
		supplied.DcapImageHashes = new(domain.DcapImageHashes)
		if err := json.Unmarshal(header.Measurements, supplied.DcapImageHashes); err != nil {
			return domain.SuppliedMeasurements{}, err
		}
		if err := supplied.DcapImageHashes.Validate(); err != nil {
			return domain.SuppliedMeasurements{}, err
		}
	case domain.ExpectedMeasurementNoAttestation:
		if len(header.Measurements) > 0 && string(header.Measurements) != "null" {
			return domain.SuppliedMeasurements{}, errors.New("no_attestation must not contain measurements")
		}
	default:
		return domain.SuppliedMeasurements{}, fmt.Errorf("unknown measurement type %q", header.Type)
	}

	return supplied, nil
}

func measurementTypeMatchesAttestationType(measurementType domain.ExpectedMeasurementType, attestationType string) bool {
	switch measurementType {
	case domain.ExpectedMeasurementDCAP:
		return attestationType == "dcap-tdx" || attestationType == "gcp-tdx"
	case domain.ExpectedMeasurementAzure:
		return attestationType == "azure-tdx"
	case domain.ExpectedMeasurementImage:
		return attestationType == "dcap-tdx" || attestationType == "gcp-tdx"
	case domain.ExpectedMeasurementNoAttestation:
		return attestationType == "none"
	default:
		return false
	}
}

func (bhs *BuilderHubHandler) getAuthData(r *http.Request) (*AuthData, error) {
	attestationType := r.Header.Get(AttestationTypeHeader)
	if attestationType == "" {
		return nil, fmt.Errorf("attestation type is empty %w", ErrInvalidAuthData)
	}
	measurementHeader := r.Header.Get(MeasurementHeader)
	measurementData, err := parseMeasurementHeader(measurementHeader, attestationType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse measurement header: %v %w", err, ErrInvalidAuthData)
	}
	ipHeaders := r.Header.Values(ForwardedHeader)
	if len(ipHeaders) == 0 {
		return nil, fmt.Errorf("ip header is empty %w", ErrInvalidAuthData)
	}
	// NOTE: we need this quite awkward logic since header comes not in the canonical format, with space.
	ipHeader := ipHeaders[len(ipHeaders)-1]
	ipHeaders = strings.Split(ipHeader, ",")
	ipHeader = ipHeaders[len(ipHeaders)-1]
	ipHeader = strings.TrimSpace(ipHeader)

	ip := net.ParseIP(ipHeader)
	if ip == nil {
		return nil, fmt.Errorf("failed to parse ip %s %w", ipHeader, ErrInvalidAuthData)
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
	pMeasurements := make([]Measurement, 0, len(measurements))
	for _, m := range measurements {
		pMeasurements = append(pMeasurements, fromDomainMeasurement(m))
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
	authData, err := bhs.getAuthData(r)
	if err != nil {
		bhs.log.Warn("malformed auth data", "error", err)
		w.WriteHeader(http.StatusForbidden)
		return
	}
	builder, _, err := bhs.builderHubService.VerifyIPAndMeasurements(r.Context(), authData.IP, authData.MeasurementData, authData.AttestationType)
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

	builders, err := bhs.builderHubService.GetActiveBuilders(r.Context(), builder.Network)
	if err != nil {
		bhs.log.Error("failed to fetch active builders from db", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	pBuilders := make([]BuilderWithServiceCreds, 0, len(builders))
	for _, b := range builders {
		pBuilders = append(pBuilders, fromDomainBuilderWithServices(b))
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
	builders, err := bhs.builderHubService.GetActiveBuilders(r.Context(), domain.ProductionNetwork)
	if err != nil {
		bhs.log.Error("failed to fetch active builders from db", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	pBuilders := make([]BuilderWithServiceCreds, 0, len(builders))
	for _, b := range builders {
		pBuilders = append(pBuilders, fromDomainBuilderWithServices(b))
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

func (bhs *BuilderHubHandler) GetActiveBuildersNoAuthNetworked(w http.ResponseWriter, r *http.Request) {
	network := chi.URLParam(r, "network")
	if network == "" {
		bhs.BadRequest(w, r, "network is empty")
		return
	}

	builders, err := bhs.builderHubService.GetActiveBuilders(r.Context(), network)
	if err != nil {
		bhs.log.Error("failed to fetch active builders from db", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	pBuilders := make([]BuilderWithServiceCreds, 0, len(builders))
	for _, b := range builders {
		pBuilders = append(pBuilders, fromDomainBuilderWithServices(b))
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
	builder, measurementName, err := bhs.builderHubService.VerifyIPAndMeasurements(r.Context(), authData.IP, authData.MeasurementData, authData.AttestationType)
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
	bts, err := bhs.builderHubService.GetConfigWithSecrets(r.Context(), builder.Name)
	if err != nil {
		bhs.log.Error("failed to get config with secrets", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// add event log
	err = bhs.builderHubService.LogEvent(r.Context(), domain.EventGetConfig, builder.Name, measurementName)
	if err != nil {
		bhs.log.Error("failed to get log event", "error", err)
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
	builder, measurementName, err := bhs.builderHubService.VerifyIPAndMeasurements(r.Context(), authData.IP, authData.MeasurementData, authData.AttestationType)
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

	service := chi.URLParam(r, "service")
	// TODO: validate service
	if service == "" {
		bhs.BadRequest(w, r, "service is empty")
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
		bhs.BadRequest(w, r, "failed to unmarshal request body", err)
		return
	}

	if sc.ECDSAPubkey == nil && sc.TLSCert == "" {
		bhs.BadRequest(w, r, "no credentials provided")
		return
	}

	tlsCert := sc.TLSCert

	var ecdsaPubkey []byte
	if sc.ECDSAPubkey != nil {
		ecdsaPubkey = sc.ECDSAPubkey.Bytes()
	}

	err = bhs.builderHubService.RegisterCredentialsForBuilder(r.Context(), builder.Name, service, tlsCert, ecdsaPubkey, measurementName, authData.AttestationType, sc.Region)
	if err != nil {
		bhs.log.Error("Failed to register credentials", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
