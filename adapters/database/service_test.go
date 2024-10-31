package database

import (
	"context"
	"fmt"
	"net"
	"testing"
)

func TestGetBuilder(t *testing.T) {
	serv, err := NewDatabaseService("postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		t.Errorf("NewDatabaseService() = %v; want nil", err)
	}
	t.Run("GetBuilder2", func(t *testing.T) {
		t.Run("should return a builder", func(t *testing.T) {
			whitelist, err := serv.GetBuilderByIP(net.ParseIP("192.168.1.1"))
			if err != nil {
				t.Errorf("GetIPWhitelistByIP() = %v; want nil", err)
			}
			fmt.Println(whitelist)
		})
		t.Run("get all active builders", func(t *testing.T) {
			whitelist, err := serv.GetActiveBuildersWithServiceCredentials(context.Background())
			if err != nil {
				t.Errorf("GetIPWhitelistByIP() = %v; want nil", err)
			}
			fmt.Println(whitelist)
		})
		t.Run("get all active measurements", func(t *testing.T) {
			whitelist, err := serv.GetActiveMeasurements(context.Background())
			if err != nil {
				t.Errorf("GetIPWhitelistByIP() = %v; want nil", err)
			}
			fmt.Println(whitelist)
		})
	})
}
