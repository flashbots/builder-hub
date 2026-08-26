package database

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"

	"github.com/flashbots/builder-hub/domain"
	"github.com/flashbots/builder-hub/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestGetBuilder(t *testing.T) {
	serv, err := NewDatabaseService(testutil.PostgresDSN(t))
	if err != nil {
		t.Errorf("NewDatabaseService() = %v; want nil", err)
	}
	_, err = serv.DB.Exec("TRUNCATE TABLE public.builders CASCADE")
	require.NoError(t, err)
	_, err = serv.DB.Exec("TRUNCATE TABLE public.measurements_whitelist CASCADE")
	require.NoError(t, err)

	t.Run("GetBuilder2", func(t *testing.T) {
		t.Run("should return a builder", func(t *testing.T) {
			_, err := serv.DB.Exec("INSERT INTO public.builders (name, ip_address, is_active, created_at, updated_at, network) VALUES ('flashbots-builder', '192.168.1.1', true, '2024-10-11 13:05:56.845615 +00:00', '2024-10-11 13:05:56.845615 +00:00', 'production');")
			require.NoError(t, err)
			whitelist, err := serv.GetBuilderByIP(net.ParseIP("192.168.1.1"))
			require.NoError(t, err)
			require.Equal(t, whitelist.Name, "flashbots-builder")
		})
		t.Run("get all active builders", func(t *testing.T) {
			whitelist, err := serv.GetActiveBuildersWithServiceCredentials(context.Background(), domain.ProductionNetwork)
			require.Nil(t, err)
			require.Lenf(t, whitelist, 1, "expected 1 builder, got %d", len(whitelist))
			require.Equal(t, whitelist[0].Builder.Name, "flashbots-builder")
		})
		t.Run("get all active measurements", func(t *testing.T) {
			whitelist, err := serv.GetActiveMeasurements(context.Background())
			if err != nil {
				t.Errorf("GetIPWhitelistByIP() = %v; want nil", err)
			}
			require.Len(t, whitelist, 0)
		})
	})
}

func TestPortableMeasurement(t *testing.T) {
	dbService, err := NewDatabaseService(testutil.PostgresDSN(t))
	require.NoError(t, err)
	_, err = dbService.DB.Exec("TRUNCATE TABLE public.measurements_whitelist CASCADE")
	require.NoError(t, err)

	hashes := &domain.DcapImageHashes{
		UKIAuthenticode:    strings.Repeat("1", 96),
		KernelAuthenticode: strings.Repeat("2", 96),
		CmdlineHash:        strings.Repeat("3", 96),
		InitrdHash:         strings.Repeat("4", 96),
		GPTDiskGUIDHash:    strings.Repeat("5", 96),
	}
	err = dbService.AddMeasurement(context.Background(), domain.Measurement{
		Name:            "portable-image",
		AttestationType: "dcap-tdx",
		DcapImageHashes: hashes,
	}, true)
	require.NoError(t, err)

	measurements, err := dbService.GetActiveMeasurementsByType(context.Background(), "dcap-tdx")
	require.NoError(t, err)
	require.Len(t, measurements, 1)
	require.Nil(t, measurements[0].Measurement)
	require.Equal(t, hashes, measurements[0].DcapImageHashes)

	t.Run("rejects missing policy representation", func(t *testing.T) {
		_, err := dbService.DB.Exec(`
			INSERT INTO measurements_whitelist (name, attestation_type, measurement, dcap_image_hashes)
			VALUES ('missing-policy', 'dcap-tdx', NULL, NULL)
		`)
		require.Error(t, err)
	})

	t.Run("rejects JSON null policy representation", func(t *testing.T) {
		_, err := dbService.DB.Exec(`
			INSERT INTO measurements_whitelist (name, attestation_type, measurement, dcap_image_hashes)
			VALUES ('json-null-policy', 'dcap-tdx', 'null'::jsonb, NULL)
		`)
		require.Error(t, err)
	})

	t.Run("rejects non-object policy representation", func(t *testing.T) {
		_, err := dbService.DB.Exec(`
			INSERT INTO measurements_whitelist (name, attestation_type, measurement, dcap_image_hashes)
			VALUES ('array-policy', 'dcap-tdx', '[]'::jsonb, NULL)
		`)
		require.Error(t, err)
	})
}

func TestAdminFlow(t *testing.T) {
	dbService, err := NewDatabaseService(testutil.PostgresDSN(t))
	if err != nil {
		t.Errorf("NewDatabaseService() = %v; want nil", err)
	}
	_, err = dbService.DB.Exec("TRUNCATE TABLE public.builders CASCADE")
	require.NoError(t, err)
	_, err = dbService.DB.Exec("TRUNCATE TABLE public.measurements_whitelist CASCADE")
	require.NoError(t, err)

	t.Run("AdminFlow", func(t *testing.T) {
		t.Run("add measurement", func(t *testing.T) {
			err := dbService.AddMeasurement(context.Background(), *domain.NewMeasurement("test-measurement-1", "test-type", map[string]domain.SingleMeasurement{"test": {Expected: "0x1234"}}), false)
			require.NoError(t, err)
		})
		t.Run("add builder", func(t *testing.T) {
			builder := domain.Builder{
				Name:      "test-builder",
				IPAddress: net.ParseIP("127.0.0.1"),
				Network:   domain.ProductionNetwork,
				IsActive:  false,
			}
			err := dbService.AddBuilder(context.Background(), builder)
			require.NoError(t, err)
		})
		t.Run("add builder config", func(t *testing.T) {
			conf := `{"key1":"value1", "key2":"value2"}`
			err := dbService.AddBuilderConfig(context.Background(), "test-builder", json.RawMessage(conf))
			require.NoError(t, err)
		})
		t.Run("get builders", func(t *testing.T) {
			builders, err := dbService.GetActiveBuildersWithServiceCredentials(context.Background(), "")
			require.NoError(t, err)
			require.Len(t, builders, 0)
		})
		t.Run("activate measurements", func(t *testing.T) {
			err := dbService.ChangeActiveStatusForMeasurement(context.Background(), "test-measurement-1", true)
			require.NoError(t, err)
		})
		t.Run("activate builder", func(t *testing.T) {
			err := dbService.ChangeActiveStatusForBuilder(context.Background(), "test-builder", true)
			require.NoError(t, err)
		})
		t.Run("get builders", func(t *testing.T) {
			builders, err := dbService.GetActiveBuildersWithServiceCredentials(context.Background(), domain.ProductionNetwork)
			require.NoError(t, err)
			require.Len(t, builders, 1)
			require.Equal(t, builders[0].Builder.Name, "test-builder")
		})
		t.Run("get measurements", func(t *testing.T) {
			measurements, err := dbService.GetActiveMeasurements(context.Background())
			require.NoError(t, err)
			require.Len(t, measurements, 1)
			require.Equal(t, measurements[0].Name, "test-measurement-1")
		})
	})
}
