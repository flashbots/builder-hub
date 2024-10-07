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
	//t.Run("GetBuilder", func(t *testing.T) {
	//	_, err = db.Exec("create temporary table t (id serial primary key, ip inet not null);")
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//
	//	var paramIP pgtype.Inet
	//	err = paramIP.Set("10.0.0.0/16")
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//
	//	_, err = db.Exec("insert into t (ip) values ($1);", paramIP)
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//
	//	var resultIP pgtype.Inet
	//	err = db.QueryRow("select ip from t").Scan(&resultIP)
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//
	//	fmt.Println(resultIP.IPNet)
	//})
	//t.Run("GetBuilder", func(t *testing.T) {
	//	t.Run("should return a builder", func(t *testing.T) {
	//		builder, err := serv.GetBuilderByIP(net.ParseIP("192.168.1.100"))
	//		if err != nil {
	//			t.Errorf("GetBuilder() = %v; want nil", err)
	//		}
	//		fmt.Println(builder)
	//	})
	//})
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
