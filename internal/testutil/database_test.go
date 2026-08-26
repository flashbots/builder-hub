package testutil

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidatePostgresDSN(t *testing.T) {
	testCases := []struct {
		name    string
		dsn     string
		wantErr bool
	}{
		{
			name: "localhost test database",
			dsn:  "postgres://postgres:postgres@localhost:5432/builder_hub_test?sslmode=disable",
		},
		{
			name: "IPv4 loopback test database",
			dsn:  "postgres://postgres:postgres@127.0.0.1:5432/builder_hub_test?sslmode=disable",
		},
		{
			name: "IPv6 loopback test database",
			dsn:  "postgres://postgres:postgres@[::1]:5432/builder_hub_test?sslmode=disable",
		},
		{
			name:    "target overridden in query",
			dsn:     "postgres://postgres:postgres@localhost:5432/builder_hub_test?host=production.example.com&dbname=production",
			wantErr: true,
		},
		{
			name:    "service configuration in query",
			dsn:     "postgres://postgres:postgres@localhost:5432/builder_hub_test?service=production",
			wantErr: true,
		},
		{
			name:    "production host",
			dsn:     "postgres://postgres:postgres@database.example.com:5432/builder_hub_test?sslmode=disable",
			wantErr: true,
		},
		{
			name:    "wrong database",
			dsn:     "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable",
			wantErr: true,
		},
		{
			name:    "wrong scheme",
			dsn:     "mysql://localhost/builder_hub_test",
			wantErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := validatePostgresDSN(testCase.dsn)
			if testCase.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

type stubDatabase struct {
	databaseName string
	err          error
}

func (s stubDatabase) Get(dest any, _ string, _ ...any) error {
	if s.err != nil {
		return s.err
	}
	*(dest.(*string)) = s.databaseName
	return nil
}

func TestValidateConnectedDatabase(t *testing.T) {
	t.Run("accepts test database", func(t *testing.T) {
		require.NoError(t, validateConnectedDatabase(stubDatabase{databaseName: TestDatabaseName}))
	})

	t.Run("rejects another database", func(t *testing.T) {
		require.Error(t, validateConnectedDatabase(stubDatabase{databaseName: "production"}))
	})

	t.Run("rejects verification failure", func(t *testing.T) {
		require.Error(t, validateConnectedDatabase(stubDatabase{err: errors.New("query failed")}))
	})
}
