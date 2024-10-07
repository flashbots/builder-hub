package application

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

type ExampleConfig struct {
	RethChainDownloader struct {
		AwsAccessKeyId     string `json:"aws_access_key_id"`
		AwsSecretAccessKey string `json:"aws_secret_access_key"`
		RcloneS3Endpoint   string `json:"rclone_s3_endpoint"`
	} `json:"reth_chain_downloader"`
	OrderflowProxy struct {
		FlashbotsOfSigningKey string   `json:"flashbots_of_signing_key"`
		BuilderPublicIp       string   `json:"builder_public_ip"`
		TlsHosts              []string `json:"tls_hosts"`
	} `json:"orderflow_proxy"`
	Rbuilder struct {
		ExtraData                string `json:"extra_data"`
		RelaySecretKey           string `json:"relay_secret_key"`
		OptimisticRelaySecretKey string `json:"optimistic_relay_secret_key"`
		CoinbaseSecretKey        string `json:"coinbase_secret_key"`
		AlwaysSeal               bool   `json:"always_seal"`
		Relays                   []struct {
			Name             string `json:"name"`
			Url              string `json:"url"`
			UseSszForSubmit  bool   `json:"use_ssz_for_submit"`
			UseGzipForSubmit bool   `json:"use_gzip_for_submit"`
			Priority         int    `json:"priority"`
			Optimistic       bool   `json:"optimistic"`
		} `json:"relays"`
	} `json:"rbuilder"`
	Prometheus struct {
		ScrapeInterval             string `json:"scrape_interval"`
		StaticConfigsDefaultLabels []struct {
			LabelKey   string `json:"label_key"`
			LabelValue string `json:"label_value"`
		} `json:"static_configs_default_labels"`
		LighthouseMetrics struct {
			Enabled bool     `json:"enabled"`
			Targets []string `json:"targets"`
		} `json:"lighthouse_metrics"`
		RethMetrics struct {
			Enabled bool     `json:"enabled"`
			Targets []string `json:"targets"`
		} `json:"reth_metrics"`
		RbuilderMetrics struct {
			Enabled bool     `json:"enabled"`
			Targets []string `json:"targets"`
		} `json:"rbuilder_metrics"`
		RemoteWrite []struct {
			Name string `json:"name"`
			Url  string `json:"url"`
		} `json:"remote_write"`
	} `json:"prometheus"`
	ProcessExporter struct {
		ProcessNames []struct {
			Name    string   `json:"name"`
			Cmdline []string `json:"cmdline"`
		} `json:"process_names"`
	} `json:"process_exporter"`
	Fluentbit struct {
		InputTags               string `json:"input_tags"`
		OutputCwLogGroupName    string `json:"output_cw_log_group_name"`
		OutputCwLogStreamPrefix string `json:"output_cw_log_stream_prefix"`
	} `json:"fluentbit"`
}

func TestMerge(t *testing.T) {
	exStr := `{
    "reth_chain_downloader": {
        "aws_access_key_id": "string",
        "aws_secret_access_key": "string",
        "rclone_s3_endpoint": "string"
    },
    "orderflow_proxy": {
        "flashbots_of_signing_key": "0x00",
        "builder_public_ip": "1.2.3.4",
        "tls_hosts": [
            "1.2.3.4",
            "fundomain.builderx.io",
            "172.27.14.1",
            "2001:db8::123.123.123.123"
        ]
    },
    "rbuilder": {
        "extra_data": "Illuminate Dmocratize Dstribute",
        "relay_secret_key": "0x00",
        "optimistic_relay_secret_key": "0x00",
        "coinbase_secret_key": "0x00",
        "always_seal": true,
        "relays": [
            {
                "name": "flashbots",
                "url": "https://0xac6e77dfe25ecd6110b8e780608cce0dab71fdd5ebea22a16c0205200f2f8e2e3ad3b71d3499c54ad14d6c21b41a37ae@boost-relay.flashbots.net",
                "use_ssz_for_submit": true,
                "use_gzip_for_submit": false,
                "priority": 0,
                "optimistic": false
            },
            {
                "name": "ultrasound",
                "url": "https://0xa1559ace749633b997cb3fdacffb890aeebdb0f5a3b6aaa7eeeaf1a38af0a8fe88b9e4b1f61f236d2e64d95733327a62@relay.ultrasound.money",
                "use_ssz_for_submit": true,
                "use_gzip_for_submit": true,
                "priority": 1,
                "optimistic": true
            }
        ]
    },
    "prometheus": {
        "scrape_interval": "10s",
        "static_configs_default_labels": [
        {
            "label_key": "flashbots_net_vendor",
            "label_value": "azure"
        },
        {
            "label_key": "flashbots_net_chain",
            "label_value": "mainnet"
        }
        ],
        "lighthouse_metrics": {
            "enabled": true,
            "targets": [
                "localhost:5054"
            ]
        },
        "reth_metrics": {
            "enabled": true,
            "targets": [
                "localhost:9001"
            ]
        },
        "rbuilder_metrics": {
            "enabled": true,
            "targets": [
                "localhost:6069"
            ]
        },
        "remote_write": [
        {
            "name": "tdx-rbuilder-collector",
            "url": "https://aps-workspaces.us-east-2.amazonaws.com/workspaces/ws-xxx/api/v1/remote_write"
        }
        ]
    },
    "process_exporter": {
        "process_names": [
          {
            "name": "lighthouse",
            "cmdline": [
              "^\\/([-.0-9a-zA-Z]+\\/)*lighthouse[-.0-9a-zA-Z]* "
            ]
          },
          {
            "name": "rbuilder",
            "cmdline": [
              "^\\/([-.0-9a-zA-Z]+\\/)*rbuilder[-.0-9a-zA-Z]* "
            ]
          },
          {
            "name": "reth",
            "cmdline": [
              "^\\/([-.0-9a-zA-Z]+\\/)*reth[-.0-9a-zA-Z]* "
            ]
          }
        ]
    },
    "fluentbit": {
        "input_tags": "tag-1 tag-2",
        "output_cw_log_group_name": "multioperator-builder",
        "output_cw_log_stream_prefix": "builder-01-"
    }
}
`
	secrets := make(map[string]string)
	secrets["orderflow_proxy.flashbots_of_signing_key"] = "test_value_1"
	newC, err := MergeConfigSecrets([]byte(exStr), secrets)
	require.NoError(t, err)

	cfg := ExampleConfig{}
	err = json.Unmarshal(newC, &cfg)
	require.Equal(t, cfg.OrderflowProxy.FlashbotsOfSigningKey, "test_value_1")
}
