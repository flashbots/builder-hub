package ports

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJson(t *testing.T) {
	b := BuilderWithServiceCreds{
		Ip: "1.2.3.4",
		ServiceCreds: map[string]Service{
			"of-proxy": {
				TlsCert:     "cert",
				EcdsaPubkey: "pubkey",
			},
			"rbuilder": {
				TlsCert:     "cert",
				EcdsaPubkey: "pubkey",
			},
		},
	}
	bts, err := json.Marshal(b)
	require.NoError(t, err)
	require.Equal(t, string(bts), `{"ip":"1.2.3.4","of-proxy":{"tls_cert":"cert","ecdsa_pubkey":"pubkey"},"rbuilder":{"tls_cert":"cert","ecdsa_pubkey":"pubkey"}}`)
}
