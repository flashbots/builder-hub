package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/flashbots/builder-hub/domain"
)

type BuilderDataAccessor interface {
	GetActiveMeasurements(ctx context.Context) ([]domain.Measurement, error)
	GetActiveBuildersWithServiceCredentials(ctx context.Context, network string) ([]domain.BuilderWithServices, error)
	GetActiveMeasurementsByType(ctx context.Context, attestationType string) ([]domain.Measurement, error)
	GetBuilderByIP(ip net.IP) (*domain.Builder, error)
	GetActiveConfigForBuilder(ctx context.Context, builderName string) (json.RawMessage, error)
	RegisterCredentialsForBuilder(ctx context.Context, builderName, service, tlsCert string, ecdsaPubKey []byte, measurementName, attestationType, region string) error
	LogEvent(ctx context.Context, eventName, builderName, name string) error
}

var ErrMissingSecret = errors.New("missing secret for builder")

type SecretAccessor interface {
	GetSecretValues(ctx context.Context, builderName string) (json.RawMessage, error)
}

type BuilderHub struct {
	dataAccessor   BuilderDataAccessor
	secretAccessor SecretAccessor
}

func NewBuilderHub(dataAccessor BuilderDataAccessor, secretAccessor SecretAccessor) *BuilderHub {
	return &BuilderHub{dataAccessor: dataAccessor, secretAccessor: secretAccessor}
}

func (b *BuilderHub) GetAllowedMeasurements(ctx context.Context) ([]domain.Measurement, error) {
	return b.dataAccessor.GetActiveMeasurements(ctx)
}

func (b *BuilderHub) GetActiveBuilders(ctx context.Context, network string) ([]domain.BuilderWithServices, error) {
	return b.dataAccessor.GetActiveBuildersWithServiceCredentials(ctx, network)
}

func (b *BuilderHub) LogEvent(ctx context.Context, eventName, builderName, name string) error {
	return b.dataAccessor.LogEvent(ctx, eventName, builderName, name)
}

func (b *BuilderHub) RegisterCredentialsForBuilder(ctx context.Context, builderName, service, tlsCert string, ecdsaPubKey []byte, measurementName, attestationType, region string) error {
	return b.dataAccessor.RegisterCredentialsForBuilder(ctx, builderName, service, tlsCert, ecdsaPubKey, measurementName, attestationType, region)
}

func (b *BuilderHub) GetConfigWithSecrets(ctx context.Context, builderName string) ([]byte, error) {
	_, err := b.dataAccessor.GetActiveConfigForBuilder(ctx, builderName)
	if err != nil {
		return nil, fmt.Errorf("failing to fetch config for builder %s %w", builderName, err)
	}
	secr, err := b.secretAccessor.GetSecretValues(ctx, builderName)
	if err != nil {
		return nil, fmt.Errorf("failing to fetch secrets for builder %s %w", builderName, err)
	}
	return secr, nil
}

func (b *BuilderHub) VerifyIPAndMeasurements(ctx context.Context, ip net.IP, measurement domain.SuppliedMeasurements, attestationType string) (*domain.Builder, string, error) {
	measurements, err := b.dataAccessor.GetActiveMeasurementsByType(ctx, attestationType)
	if err != nil {
		return nil, "", fmt.Errorf("failing to fetch corresponding measurement data %s %w", attestationType, err)
	}
	measurementName, err := validateMeasurement(measurement, measurements)
	if err != nil {
		return nil, "", fmt.Errorf("failing to validate measurement %w", err)
	}

	builder, err := b.dataAccessor.GetBuilderByIP(ip)
	if err != nil {
		// TODO: might avoid logging ip though it should be ok, at least keep it for development state
		return nil, "", fmt.Errorf("failing to fetch builder by ip %s %w", ip.String(), err)
	}
	return builder, measurementName, nil
}

func validateMeasurement(measurement domain.SuppliedMeasurements, measurementTemplate []domain.Measurement) (string, error) {
	for _, m := range measurementTemplate {
		matched := false
		if measurement.Type == domain.ExpectedMeasurementLegacy {
			matched = checkMeasurement(measurement.Legacy, m)
		} else {
			matched = checkExpectedMeasurement(measurement, m)
		}
		if matched {
			return m.Name, nil
		}
	}
	return "", domain.ErrNotFound
}

func checkExpectedMeasurement(measurement domain.SuppliedMeasurements, template domain.Measurement) bool {
	expectedType, ok := expectedMeasurementType(template)
	if !ok || measurement.Type != expectedType {
		return false
	}

	switch expectedType {
	case domain.ExpectedMeasurementDCAP, domain.ExpectedMeasurementAzure:
		supplied, ok := normalizeSuppliedRegisters(measurement.Registers, template.AttestationType)
		if !ok {
			return false
		}
		expected, ok := normalizeExpectedRegisters(template.Measurement, template.AttestationType)
		return ok && equalRegisterSets(supplied, expected)
	case domain.ExpectedMeasurementImage:
		return equalImageHashes(measurement.DcapImageHashes, template.DcapImageHashes)
	case domain.ExpectedMeasurementNoAttestation:
		return true
	default:
		return false
	}
}

func expectedMeasurementType(template domain.Measurement) (domain.ExpectedMeasurementType, bool) {
	if template.DcapImageHashes != nil {
		if template.AttestationType == "dcap-tdx" || template.AttestationType == "gcp-tdx" {
			return domain.ExpectedMeasurementImage, true
		}
		return "", false
	}

	switch template.AttestationType {
	case "dcap-tdx", "gcp-tdx":
		return domain.ExpectedMeasurementDCAP, true
	case "azure-tdx":
		return domain.ExpectedMeasurementAzure, true
	case "none":
		return domain.ExpectedMeasurementNoAttestation, true
	default:
		return "", false
	}
}

func normalizeSuppliedRegisters(registers map[string][]string, attestationType string) (map[string]map[string]struct{}, bool) {
	normalized := make(map[string]map[string]struct{}, len(registers))
	for key, values := range registers {
		canonical, ok := canonicalRegisterKey(key, attestationType)
		if !ok || len(values) == 0 {
			return nil, false
		}
		if _, duplicate := normalized[canonical]; duplicate {
			return nil, false
		}
		normalized[canonical] = stringSet(values)
	}
	return normalized, true
}

func normalizeExpectedRegisters(registers map[string]domain.SingleMeasurement, attestationType string) (map[string]map[string]struct{}, bool) {
	normalized := make(map[string]map[string]struct{}, len(registers))
	for key, measurement := range registers {
		canonical, ok := canonicalRegisterKey(key, attestationType)
		values := measurement.GetExpectedValues()
		if !ok || len(values) == 0 {
			return nil, false
		}
		if _, duplicate := normalized[canonical]; duplicate {
			return nil, false
		}
		normalized[canonical] = stringSet(values)
	}
	return normalized, true
}

func canonicalRegisterKey(key, attestationType string) (string, bool) {
	switch attestationType {
	case "dcap-tdx", "gcp-tdx":
		if index, err := strconv.Atoi(key); err == nil {
			if index >= 0 && index <= 4 {
				return strconv.Itoa(index), true
			}
			return "", false
		}
		switch strings.ToLower(key) {
		case "mrtd":
			return "0", true
		case "rtmr0":
			return "1", true
		case "rtmr1":
			return "2", true
		case "rtmr2":
			return "3", true
		case "rtmr3":
			return "4", true
		default:
			return "", false
		}
	case "azure-tdx":
		indexString := key
		if len(key) >= 3 && strings.EqualFold(key[:3], "pcr") {
			indexString = key[3:]
		}
		index, err := strconv.Atoi(indexString)
		if err != nil || index < 0 || index > 23 {
			return "", false
		}
		return strconv.Itoa(index), true
	default:
		return "", false
	}
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[strings.ToLower(value)] = struct{}{}
	}
	return set
}

func equalRegisterSets(left, right map[string]map[string]struct{}) bool {
	if len(left) != len(right) {
		return false
	}
	for key, leftValues := range left {
		rightValues, ok := right[key]
		if !ok || len(leftValues) != len(rightValues) {
			return false
		}
		for value := range leftValues {
			if _, ok := rightValues[value]; !ok {
				return false
			}
		}
	}
	return true
}

func equalImageHashes(left, right *domain.DcapImageHashes) bool {
	if left == nil || right == nil {
		return false
	}
	return strings.EqualFold(left.UKIAuthenticode, right.UKIAuthenticode) &&
		strings.EqualFold(left.KernelAuthenticode, right.KernelAuthenticode) &&
		strings.EqualFold(left.CmdlineHash, right.CmdlineHash) &&
		strings.EqualFold(left.InitrdHash, right.InitrdHash) &&
		strings.EqualFold(left.GPTDiskGUIDHash, right.GPTDiskGUIDHash)
}

// validates that all fields from measurementTemplate are the same in measurement.
// For each field, the measurement value must match at least one of the expected values (OR semantics).
func checkMeasurement(measurement map[string]string, measurementTemplate domain.Measurement) bool {
	for k, v := range measurementTemplate.Measurement {
		val, ok := measurement[k]
		if !ok {
			return false
		}
		if !matchesAnyExpected(val, v.GetExpectedValues()) {
			return false
		}
	}
	return true
}

// matchesAnyExpected returns true if the value matches any of the expected values.
func matchesAnyExpected(value string, expected []string) bool {
	for _, exp := range expected {
		if value == exp {
			return true
		}
	}
	return false
}
