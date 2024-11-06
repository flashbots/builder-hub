package database

import (
	"context"
	"net"
	"os"
	"testing"

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
