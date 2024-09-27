package httpserver

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/flashbots/builder-hub/metrics"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
	"go.uber.org/atomic"
)

type HTTPServerConfig struct {
	ListenAddr  string
	MetricsAddr string
	EnablePprof bool
	Log         *httplog.Logger

	DrainDuration            time.Duration
	GracefulShutdownDuration time.Duration
	ReadTimeout              time.Duration
	WriteTimeout             time.Duration
}

type Server struct {
	cfg     *HTTPServerConfig
	isReady atomic.Bool
	log     *httplog.Logger

	srv        *http.Server
	metricsSrv *metrics.MetricsServer

	mockGetConfigResponse       string
	mockGetBuildersResponse     string
	mockGetMeasurementsResponse string
}

func NewHTTPServer(cfg *HTTPServerConfig) (srv *Server, err error) {
	srv = &Server{
		cfg:        cfg,
		log:        cfg.Log,
		srv:        nil,
		metricsSrv: metrics.NewMetricsServer(cfg.MetricsAddr, nil),
	}
	srv.isReady.Swap(true)

	srv.srv = &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      srv.getRouter(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	return srv, nil
}

func (srv *Server) getRouter() http.Handler {
	mux := chi.NewRouter()

	mux.Use(httplog.RequestLogger(srv.log))
	mux.Use(middleware.Recoverer)

	// System API
	mux.Get("/livez", srv.handleLivenessCheck)
	mux.Get("/readyz", srv.handleReadinessCheck)
	mux.Get("/drain", srv.handleDrain)
	mux.Get("/undrain", srv.handleUndrain)

	// Dev
	mux.Get("/test-panic", srv.handleTestPanic)

	// BuilderConfigHub API: https://www.notion.so/flashbots/BuilderConfigHub-1076b4a0d8768074bcdcd1c06c26ec87?pvs=4#10a6b4a0d87680fd81e0cad9bac3b8c5
	mux.Get("/api/l1-builder/v1/measurements", srv.handleGetMeasurements)
	mux.Get("/api/l1-builder/v1/configuration", srv.handleGetConfiguration)
	mux.Get("/api/l1-builder/v1/builders", srv.handleGetBuilders)
	mux.Post("/api/l1-builder/v1/register_credentials/{service}", srv.handleRegisterCredentials)

	if srv.cfg.EnablePprof {
		srv.log.Info("pprof API enabled")
		mux.Mount("/debug", middleware.Profiler())
	}

	return mux
}

func (srv *Server) _stringFromFile(fn string) (string, error) {
	content, err := os.ReadFile(fn)
	if err != nil {
		srv.log.Error("Failed to read mock response", "file", fn, "err", err)
		return "", err
	}
	return string(content), nil
}

func (srv *Server) LoadMockResponses() (err error) {
	srv.mockGetConfigResponse, err = srv._stringFromFile("testdata/get-configuration.json")
	if err != nil {
		return err
	}
	srv.mockGetBuildersResponse, err = srv._stringFromFile("testdata/get-builders.json")
	if err != nil {
		return err
	}
	srv.mockGetMeasurementsResponse, err = srv._stringFromFile("testdata/get-measurements.json")
	return err
}

func (srv *Server) RunInBackground() {
	// metrics
	if srv.cfg.MetricsAddr != "" {
		go func() {
			srv.log.With("metricsAddress", srv.cfg.MetricsAddr).Info("Starting metrics server")
			err := srv.metricsSrv.Start()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				srv.log.Error("HTTP server failed", "err", err)
			}
		}()
	}

	// api
	go func() {
		srv.log.Info("Starting HTTP server", "listenAddress", srv.cfg.ListenAddr)
		if err := srv.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srv.log.Error("HTTP server failed", "err", err)
		}
	}()
}

func (srv *Server) Shutdown() {
	// api
	ctx, cancel := context.WithTimeout(context.Background(), srv.cfg.GracefulShutdownDuration)
	defer cancel()
	if err := srv.srv.Shutdown(ctx); err != nil {
		srv.log.Error("Graceful HTTP server shutdown failed", "err", err)
	} else {
		srv.log.Info("HTTP server gracefully stopped")
	}

	// metrics
	if len(srv.cfg.MetricsAddr) != 0 {
		ctx, cancel := context.WithTimeout(context.Background(), srv.cfg.GracefulShutdownDuration)
		defer cancel()

		if err := srv.metricsSrv.Shutdown(ctx); err != nil {
			srv.log.Error("Graceful metrics server shutdown failed", "err", err)
		} else {
			srv.log.Info("Metrics server gracefully stopped")
		}
	}
}
