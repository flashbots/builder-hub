package database

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"testing"

	"github.com/flashbots/builder-hub/domain"
	"github.com/stretchr/testify/require"
)

func TestGetBuilder(t *testing.T) {
	if os.Getenv("RUN_DB_TESTS") != "1" {
		t.Skip("skipping test; RUN_DB_TESTS is not set to 1")
	}
	serv, err := NewDatabaseService("postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		t.Errorf("NewDatabaseService() = %v; want nil", err)
	}
	t.Run("GetBuilder2", func(t *testing.T) {
		t.Run("should return a builder", func(t *testing.T) {
			_, err := serv.DB.Exec("INSERT INTO public.builders (name, ip_address, is_active, created_at, updated_at) VALUES ('flashbots-builder', '192.168.1.1', true, '2024-10-11 13:05:56.845615 +00:00', '2024-10-11 13:05:56.845615 +00:00');")
			require.NoError(t, err)
			whitelist, err := serv.GetBuilderByIP(net.ParseIP("192.168.1.1"))
			require.NoError(t, err)
			require.Equal(t, whitelist.Name, "flashbots-builder")
		})
		t.Run("get all active builders", func(t *testing.T) {
			whitelist, err := serv.GetActiveBuildersWithServiceCredentials(context.Background())
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

func TestAdminFlow(t *testing.T) {
	//if os.Getenv("RUN_DB_TESTS") != "1" {
	//	t.Skip("skipping test; RUN_DB_TESTS is not set to 1")
	//}
	dbService, err := NewDatabaseService("postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
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
			builders, err := dbService.GetActiveBuildersWithServiceCredentials(context.Background())
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
			builders, err := dbService.GetActiveBuildersWithServiceCredentials(context.Background())
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
