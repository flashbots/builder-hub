// Package common contains common utilities and functions used by the service.
package common

import (
	"log/slog"

	"github.com/go-chi/httplog/v2"
)

type LoggingOpts struct {
	Service        string
	JSON           bool
	Debug          bool
	Concise        bool
	RequestHeaders bool
	Tags           map[string]string
}

func SetupLogger(opts *LoggingOpts) (log *httplog.Logger) {
	logLevel := slog.LevelInfo
	if opts.Debug {
		logLevel = slog.LevelDebug
	}

	logger := httplog.NewLogger(opts.Service, httplog.Options{
		JSON:           opts.JSON,
		LogLevel:       logLevel,
		Concise:        opts.Concise,
		RequestHeaders: opts.RequestHeaders,
		Tags:           opts.Tags,
	})
	return logger
}
