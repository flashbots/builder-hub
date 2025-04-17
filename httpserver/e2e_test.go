package httpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/flashbots/builder-hub/adapters/database"
	"github.com/flashbots/builder-hub/application"
	"github.com/flashbots/builder-hub/domain"
	"github.com/flashbots/builder-hub/ports"
	"github.com/stretchr/testify/require"
)

func TestCreateMultipleBuilders(t *testing.T) {
	if os.Getenv("RUN_DB_TESTS") != "1" {
		t.Skip("skipping test; RUN_DB_TESTS is not set to 1")
	}
	s, _, _ := createServer(t)

	createBuilder(t, s, "test-builder-1", "127.0.0.1", domain.ProductionNetwork)
	createBuilder(t, s, "test-builder-2", "127.0.0.2", domain.ProductionNetwork)
	createBuilder(t, s, "test-builder-3", "127.0.0.3", domain.ProductionNetwork)
}

func TestCreateMultipleNetworkBuilders(t *testing.T) {
	if os.Getenv("RUN_DB_TESTS") != "1" {
		t.Skip("skipping test; RUN_DB_TESTS is not set to 1")
	}
	s, _, _ := createServer(t)

	createBuilder(t, s, "production-1", "127.0.0.1", domain.ProductionNetwork)
	createBuilder(t, s, "production-2", "127.0.0.2", domain.ProductionNetwork)
	createBuilder(t, s, "production-3", "127.0.0.3", domain.ProductionNetwork)

	createBuilder(t, s, "test-1", "127.0.1.1", "test-network")
	createBuilder(t, s, "test-2", "127.0.1.2", "test-network")
	createBuilder(t, s, "test-3", "127.0.1.3", "test-network")
	createBuilder(t, s, "test-4", "127.0.1.4", "test-network")

	measurement := ports.Measurement{
		Name:            "test-measurement-1",
		AttestationType: "test-attestation-type-1",
		Measurements: map[string]domain.SingleMeasurement{
			"8": {
				Expected: "0000000000000000000000000000000000000000000000000000000000000000",
			},
			"11": {
				Expected: "efa43e0beff151b0f251c4abf48152382b1452b4414dbd737b4127de05ca31f7",
			},
		},
	}
	createMeasurement(t, s, measurement)

	t.Run("NoAuthV1", func(t *testing.T) {
		var resp []ports.BuilderWithServiceCreds
		sc, _ := execRequestNoAuth(t, s.GetInternalRouter(), http.MethodGet, "/api/internal/l1-builder/v1/builders", nil, &resp)
		require.Equal(t, http.StatusOK, sc)
		require.Len(t, resp, 3)
		validNames := []string{"production-1", "production-2", "production-3"}
		for _, b := range resp {
			if !slices.Contains(validNames, b.Name) {
				require.Fail(t, "Builder not found")
			}
		}
	})

	t.Run("NoAuthV2 Networked", func(t *testing.T) {
		var resp []ports.BuilderWithServiceCreds
		sc, _ := execRequestNoAuth(t, s.GetInternalRouter(), http.MethodGet, fmt.Sprintf("/api/internal/l1-builder/v2/network/%s/builders", domain.ProductionNetwork), nil, &resp)
		require.Equal(t, http.StatusOK, sc)
		require.Len(t, resp, 3)
		validNames := []string{"production-1", "production-2", "production-3"}
		for _, b := range resp {
			if !slices.Contains(validNames, b.Name) {
				require.Fail(t, "Builder not found")
			}
		}
	})

	t.Run("NoAuthV2 Networked other network", func(t *testing.T) {
		var resp []ports.BuilderWithServiceCreds
		sc, _ := execRequestNoAuth(t, s.GetInternalRouter(), http.MethodGet, fmt.Sprintf("/api/internal/l1-builder/v2/network/%s/builders", "test-network"), nil, &resp)
		require.Equal(t, http.StatusOK, sc)
		require.Len(t, resp, 4)
		validNames := []string{"test-1", "test-2", "test-3", "test-4"}
		for _, b := range resp {
			if !slices.Contains(validNames, b.Name) {
				require.Fail(t, "Builder not found")
			}
		}
	})

	t.Run("Auth active builders prod", func(t *testing.T) {
		resp := make([]ports.BuilderWithServiceCreds, 0)
		status, _ := execRequestAuth(t, s.GetRouter(), http.MethodGet, "/api/l1-builder/v1/builders", nil, &resp, measurement.AttestationType, map[string]string{"8": "0000000000000000000000000000000000000000000000000000000000000000", "11": "efa43e0beff151b0f251c4abf48152382b1452b4414dbd737b4127de05ca31f7"}, "127.0.0.1")
		require.Equal(t, http.StatusOK, status)
		require.Len(t, resp, 3)
		validNames := []string{"production-1", "production-2", "production-3"}
		for _, b := range resp {
			if !slices.Contains(validNames, b.Name) {
				require.Fail(t, "Builder not found")
			}
		}
	})

	t.Run("Auth active builders other network", func(t *testing.T) {
		resp := make([]ports.BuilderWithServiceCreds, 0)
		status, _ := execRequestAuth(t, s.GetRouter(), http.MethodGet, "/api/l1-builder/v1/builders", nil, &resp, measurement.AttestationType, map[string]string{"8": "0000000000000000000000000000000000000000000000000000000000000000", "11": "efa43e0beff151b0f251c4abf48152382b1452b4414dbd737b4127de05ca31f7"}, "127.0.1.1")
		require.Equal(t, http.StatusOK, status)
		require.Len(t, resp, 4)
		validNames := []string{"test-1", "test-2", "test-3", "test-4"}
		for _, b := range resp {
			if !slices.Contains(validNames, b.Name) {
				require.Fail(t, "Builder not found")
			}
		}
	})
}

func TestCreateMultipleMeasurements(t *testing.T) {
	if os.Getenv("RUN_DB_TESTS") != "1" {
		t.Skip("skipping test; RUN_DB_TESTS is not set to 1")
	}
	s, _, _ := createServer(t)

	createMeasurement(t, s, ports.Measurement{
		Name:            "test-measurement-1",
		AttestationType: "test-attestation-type-1",
		Measurements: map[string]domain.SingleMeasurement{
			"8": {
				Expected: "0000000000000000000000000000000000000000000000000000000000000000",
			},
			"11": {
				Expected: "efa43e0beff151b0f251c4abf48152382b1452b4414dbd737b4127de05ca31f7",
			},
		},
	})
	createMeasurement(t, s, ports.Measurement{
		Name:            "test-measurement-2",
		AttestationType: "test-attestation-type-2",
		Measurements: map[string]domain.SingleMeasurement{
			"8": {
				Expected: "0000000000000000000000000000000000000000000000000000000000000000",
			},
			"11": {
				Expected: "efa43e0beff151b0f251c4abf48152382b1452b4414dbd737b4127de05ca31f8",
			},
		},
	})
}

func TestAuthInteractionFlow(t *testing.T) {
	if os.Getenv("RUN_DB_TESTS") != "1" {
		t.Skip("skipping test; RUN_DB_TESTS is not set to 1")
	}

	s, _, _ := createServer(t)

	builderName := "test_builder_1"
	ip := "127.0.0.1"
	createBuilder(t, s, builderName, ip, domain.ProductionNetwork)
	measurement := ports.Measurement{
		Name:            "test-measurement-1",
		AttestationType: "test-attestation-type-1",
		Measurements: map[string]domain.SingleMeasurement{
			"8": {
				Expected: "0000000000000000000000000000000000000000000000000000000000000000",
			},
			"11": {
				Expected: "efa43e0beff151b0f251c4abf48152382b1452b4414dbd737b4127de05ca31f7",
			},
		},
	}
	createMeasurement(t, s, measurement)

	t.Run("GetConfig", func(t *testing.T) {
		resp := make(map[string]string)
		status, _ := execRequestAuth(t, s.GetRouter(), http.MethodGet, "/api/l1-builder/v1/configuration", nil, &resp, measurement.AttestationType, map[string]string{"8": "0000000000000000000000000000000000000000000000000000000000000000", "11": "efa43e0beff151b0f251c4abf48152382b1452b4414dbd737b4127de05ca31f7"}, "127.0.0.1")
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, builderName+"_test_value_1", resp["test_key_1"])
		require.Equal(t, builderName+"_test_secret_value", resp["test_secret_1"])
	})
	t.Run("GetConfigFailureWrongIP", func(t *testing.T) {
		resp := make(map[string]string)
		status, _ := execRequestAuth(t, s.GetRouter(), http.MethodGet, "/api/l1-builder/v1/configuration", nil, &resp, measurement.AttestationType, map[string]string{"8": "0000000000000000000000000000000000000000000000000000000000000000", "11": "efa43e0beff151b0f251c4abf48152382b1452b4414dbd737b4127de05ca31f7"}, "127.0.0.2")
		require.Equal(t, http.StatusForbidden, status)
	})
	t.Run("GetConfigFailureWrongMeasurement", func(t *testing.T) {
		resp := make(map[string]string)
		status, _ := execRequestAuth(t, s.GetRouter(), http.MethodGet, "/api/l1-builder/v1/configuration", nil, &resp, measurement.AttestationType, map[string]string{"8": "0000000000000000000000000000000000000000000000000000000000000000", "11": "abc"}, "127.0.0.1")
		require.Equal(t, http.StatusForbidden, status)
	})
	t.Run("GetConfigFailureWrongMeasurementType", func(t *testing.T) {
		resp := make(map[string]string)
		status, _ := execRequestAuth(t, s.GetRouter(), http.MethodGet, "/api/l1-builder/v1/configuration", nil, &resp, "wrong-type", map[string]string{"8": "0000000000000000000000000000000000000000000000000000000000000000", "11": "efa43e0beff151b0f251c4abf48152382b1452b4414dbd737b4127de05ca31f7"}, "127.0.0.1")
		require.Equal(t, http.StatusForbidden, status)
	})

	t.Run("RegisterCredentials", func(t *testing.T) {
		addr := common.HexToAddress("0x1234567890123456789012345678901234567890")
		sc := ports.ServiceCred{
			TLSCert:     "test-cert-no-validation",
			ECDSAPubkey: &addr,
		}
		status, _ := execRequestAuth(t, s.GetRouter(), http.MethodPost, "/api/l1-builder/v1/register_credentials/rbuilder", sc, nil, measurement.AttestationType, map[string]string{"8": "0000000000000000000000000000000000000000000000000000000000000000", "11": "efa43e0beff151b0f251c4abf48152382b1452b4414dbd737b4127de05ca31f7"}, "127.0.0.1")
		require.Equal(t, http.StatusOK, status)
	})
	t.Run("CheckCredentials", func(t *testing.T) {
		resp := make([]ports.BuilderWithServiceCreds, 0)
		sc, bts := execRequestAuth(t, s.GetRouter(), http.MethodGet, "/api/l1-builder/v1/builders", nil, &resp, measurement.AttestationType, map[string]string{"8": "0000000000000000000000000000000000000000000000000000000000000000", "11": "efa43e0beff151b0f251c4abf48152382b1452b4414dbd737b4127de05ca31f7"}, "127.0.0.1")
		c := string(bts)
		fmt.Println(c)
		require.Equal(t, http.StatusOK, sc)
		require.Equal(t, 1, len(resp))
		require.Equal(t, builderName, resp[0].Name)
		require.Equal(t, "test-cert-no-validation", resp[0].ServiceCreds["rbuilder"].TLSCert)
		require.Equal(t, "0x1234567890123456789012345678901234567890", resp[0].ServiceCreds["rbuilder"].ECDSAPubkey.String())
	})
}

// createBuilder emulates admin flow to provision a builder
func createBuilder(t *testing.T, s *Server, builderName, ip, network string) {
	builder := ports.Builder{
		Name:      builderName,
		IPAddress: ip,
		Network:   network,
	}
	t.Run("CreateBuilder", func(t *testing.T) {
		sc, _ := execRequestNoAuth(t, s.GetAdminRouter(), http.MethodPost, "/api/admin/v1/builders", builder, nil)
		require.Equal(t, http.StatusOK, sc)
	})

	t.Run("CreateBuilderConfiguration", func(t *testing.T) {
		builderConf := map[string]string{"test_key_1": builderName + "_test_value_1"}
		sc, _ := execRequestNoAuth(t, s.GetAdminRouter(), http.MethodPost, "/api/admin/v1/builders/configuration/"+builderName, builderConf, nil)
		require.Equal(t, http.StatusOK, sc)
	})
	t.Run("SetSecrets", func(t *testing.T) {
		sec := map[string]string{"test_secret_1": builderName + "_test_secret_value"}
		sc, _ := execRequestNoAuth(t, s.GetAdminRouter(), http.MethodPost, "/api/admin/v1/builders/secrets/"+builderName, sec, nil)
		require.Equal(t, http.StatusOK, sc)
	})

	t.Run("ActivateBuilder", func(t *testing.T) {
		activate := ports.ActivationRequest{Enabled: true}
		sc, _ := execRequestNoAuth(t, s.GetAdminRouter(), http.MethodPost, "/api/admin/v1/builders/activation/"+builderName, activate, nil)
		require.Equal(t, http.StatusOK, sc)
	})

	t.Run("CheckBuilder", func(t *testing.T) {
		// Check if the builder is created
		var resp []ports.BuilderWithServiceCreds
		sc, _ := execRequestNoAuth(t, s.GetInternalRouter(), http.MethodGet, fmt.Sprintf("/api/internal/l1-builder/v2/network/%s/builders", network), nil, &resp)
		require.Equal(t, http.StatusOK, sc)
		// require.Len(t, resp, 1)
		for _, b := range resp {
			if b.Name == builderName {
				require.Equal(t, ip, b.IP)
				return
			}
		}
		require.Fail(t, "Builder not found")
	})
	t.Run("CheckBuilderConfiguration", func(t *testing.T) {
		// Check if the builder configuration is created
		var resp map[string]string
		url := fmt.Sprintf("/api/admin/v1/builders/configuration/%s/full", builderName)
		sc, _ := execRequestNoAuth(t, s.GetAdminRouter(), http.MethodGet, url, nil, &resp)
		require.Equal(t, http.StatusOK, sc)
		require.Equal(t, builderName+"_test_value_1", resp["test_key_1"])
		require.Equal(t, builderName+"_test_secret_value", resp["test_secret_1"])
	})
}

func createMeasurement(t *testing.T, s *Server, measurement ports.Measurement) {
	t.Run("CreateMeasurement", func(t *testing.T) {
		sc, _ := execRequestNoAuth(t, s.GetAdminRouter(), http.MethodPost, "/api/admin/v1/measurements", measurement, nil)
		require.Equal(t, http.StatusOK, sc)
	})
	t.Run("ActivateMeasurement", func(t *testing.T) {
		activate := ports.ActivationRequest{Enabled: true}
		sc, _ := execRequestNoAuth(t, s.GetAdminRouter(), http.MethodPost, "/api/admin/v1/measurements/activation/"+measurement.Name, activate, nil)
		require.Equal(t, http.StatusOK, sc)
	})
	t.Run("CheckMeasurement", func(t *testing.T) {
		// Check if the measurement is created
		var resp []ports.Measurement
		sc, _ := execRequestNoAuth(t, s.GetInternalRouter(), http.MethodGet, "/api/l1-builder/v1/measurements", nil, &resp)
		require.Equal(t, http.StatusOK, sc)
		for _, m := range resp {
			if m.Name == measurement.Name {
				require.Equal(t, measurement.Name, m.Name)
				return
			}
		}
		require.Fail(t, "Measurement not found")
	})
}

func createDbService(t *testing.T) *database.Service {
	t.Helper()
	dbService, err := database.NewDatabaseService("postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		t.Errorf("NewDatabaseService() = %v; want nil", err)
	}
	_, err = dbService.DB.Exec("TRUNCATE TABLE public.builders CASCADE")
	require.NoError(t, err)
	_, err = dbService.DB.Exec("TRUNCATE TABLE public.measurements_whitelist CASCADE")
	require.NoError(t, err)
	return dbService
}

func createServer(t *testing.T) (*Server, *database.Service, *domain.InmemorySecretService) {
	t.Helper()
	dbService := createDbService(t)
	mss := domain.NewMockSecretService()
	bhs := application.NewBuilderHub(dbService, mss)
	bhh := ports.NewBuilderHubHandler(bhs, getTestLogger())
	_ = bhh
	ah := ports.NewAdminHandler(dbService, mss, getTestLogger())
	_ = ah
	s, err := NewHTTPServer(&HTTPServerConfig{
		DrainDuration: latency,
		ListenAddr:    listenAddr,
		InternalAddr:  internalAddr,
		AdminAddr:     adminAddr,
		Log:           getTestLogger(),
	}, bhh, ah)
	require.NoError(t, err)
	return s, dbService, mss
}

func execRequestNoAuth(t *testing.T, router http.Handler, method, url string, request, response any) (statusCode int, responsePayload []byte) {
	t.Helper()
	return execRequestAuth(t, router, method, url, request, response, "", nil, "")
}

func execRequestAuth(t *testing.T, router http.Handler, method, url string, request, response any, attestationType string, measurement map[string]string, ip string) (statusCode int, responsePayload []byte) {
	t.Helper()
	rBody, err := json.Marshal(request)
	require.NoError(t, err)
	tr := httptest.NewRequest(method, url, bytes.NewBuffer(rBody))
	require.NoError(t, err)
	if attestationType != "" {
		tr.Header.Set(ports.AttestationTypeHeader, attestationType)
	}
	if len(measurement) > 0 {
		measB, err := json.Marshal(measurement)
		require.NoError(t, err)
		tr.Header.Set(ports.MeasurementHeader, string(measB))
	}
	if ip != "" {
		tr.Header.Set(ports.ForwardedHeader, ip)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, tr)

	responseBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err)

	if response != nil && rr.Code >= 200 && rr.Code < 300 {
		err := json.Unmarshal(responseBody, response)
		require.NoError(t, err)
	}
	return rr.Code, responseBody
}
