// Package metrics contains all application-logic metrics
package metrics

import (
	"github.com/VictoriaMetrics/metrics"
)

var testMetric = metrics.NewCounter("test1")

// sbundleProcessDurationSummary        = metrics.NewSummary("sbundle_process_duration_milliseconds")

// const (
// 	sbundleRPCCallDurationLabel     = `sbundle_rpc_call_duration_milliseconds{method="%s"}`
// 	sbundleRPCCallErrorCounterLabel = `sbundle_rpc_call_error_total{method="%s"}`

// 	sbundleSentToBuilderLabel                = `bundle_sent_to_builder_total{builder="%s"}`
// 	sbundleSentToBuilderFailureLabel         = `bundle_sent_to_builder_failure_total{builder="%s"}`
// 	sbundleSentToBuilderDurationSummaryLabel = `bundle_sent_to_builder_duration_milliseconds{builder="%s"}`
// )

func IncSbundlesReceived() {
	testMetric.Inc()
}
