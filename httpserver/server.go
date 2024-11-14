package httpserver

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/flashbots/builder-hub/metrics"
	"github.com/flashbots/builder-hub/ports"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
	"go.uber.org/atomic"
)

type HTTPServerConfig struct {
	ListenAddr   string
	MetricsAddr  string
	AdminAddr    string
	InternalAddr string
	EnablePprof  bool
	Log          *httplog.Logger

	DrainDuration            time.Duration
	GracefulShutdownDuration time.Duration
	ReadTimeout              time.Duration
	WriteTimeout             time.Duration
}

type Server struct {
	cfg          *HTTPServerConfig
	isReady      atomic.Bool
	log          *httplog.Logger
	appHandler   *ports.BuilderHubHandler
	adminHandler *ports.AdminHandler

	srv         *http.Server
	adminSrv    *http.Server
	internalSrv *http.Server
	metricsSrv  *metrics.MetricsServer

	mockGetConfigResponse       string
	mockGetBuildersResponse     string
	mockGetMeasurementsResponse string
}

func NewHTTPServer(cfg *HTTPServerConfig, appHandler *ports.BuilderHubHandler, adminHandler *ports.AdminHandler) (srv *Server, err error) {
	srv = &Server{
		cfg:          cfg,
		log:          cfg.Log,
		appHandler:   appHandler,
		adminHandler: adminHandler,
		srv:          nil,
		metricsSrv:   metrics.NewMetricsServer(cfg.MetricsAddr, nil),
	}
	srv.isReady.Swap(true)

	srv.srv = &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      srv.getRouter(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}
	srv.internalSrv = &http.Server{
		Addr:         cfg.InternalAddr,
		Handler:      srv.getInternalRouter(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}
	srv.adminSrv = &http.Server{
		Addr:         cfg.AdminAddr,
		Handler:      srv.getAdminRouter(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	return srv, nil
}

func (srv *Server) getRouter() http.Handler {
	mux := chi.NewRouter()

	mux.Use(httplog.RequestLogger(srv.log))
	mux.Use(middleware.Recoverer)
	mux.Use(metrics.Middleware)

	// System API
	mux.Get("/livez", srv.handleLivenessCheck)
	mux.Get("/readyz", srv.handleReadinessCheck)
	mux.Get("/drain", srv.handleDrain)
	mux.Get("/undrain", srv.handleUndrain)

	mux.Get("/api/l1-builder/v1/measurements", srv.appHandler.GetAllowedMeasurements)
	mux.Get("/api/l1-builder/v1/configuration", srv.appHandler.GetConfigSecrets)
	mux.Get("/api/l1-builder/v1/builders", srv.appHandler.GetActiveBuilders)
	mux.Post("/api/l1-builder/v1/register_credentials/{service}", srv.appHandler.RegisterCredentials)
	mux.Get("/api/internal/l1-builder/v1/builders", srv.appHandler.GetActiveBuildersNoAuth)
	if srv.cfg.EnablePprof {
		srv.log.Info("pprof API enabled")
		mux.Mount("/debug", middleware.Profiler())
	}

	return mux
}

func (srv *Server) getAdminRouter() http.Handler {
	mux := chi.NewRouter()

	mux.Use(httplog.RequestLogger(srv.log))
	mux.Use(middleware.Recoverer)
	mux.Use(metrics.Middleware)

	mux.Get("/api/admin/v1/builders/configuration/{builderName}/active", srv.adminHandler.GetActiveConfigForBuilder)
	mux.Get("/api/admin/v1/builders/configuration/{builderName}/full", srv.adminHandler.GetFullConfigForBuilder)
	mux.Post("/api/admin/v1/measurements", srv.adminHandler.AddMeasurement)
	mux.Post("/api/admin/v1/builders", srv.adminHandler.AddBuilder)
	mux.Post("/api/admin/v1/builders/activation/{builderName}", srv.adminHandler.ChangeActiveStatusForBuilder)
	mux.Post("/api/admin/v1/measurements/activation/{measurementName}", srv.adminHandler.ChangeActiveStatusForMeasurement)
	mux.Post("/api/admin/v1/builders/configuration/{builderName}", srv.adminHandler.AddBuilderConfig)
	mux.Post("/api/admin/v1/builders/secrets/{builderName}", srv.adminHandler.SetSecrets)

	return mux
}

func (srv *Server) getInternalRouter() http.Handler {
	mux := chi.NewRouter()

	mux.Use(httplog.RequestLogger(srv.log))
	mux.Use(middleware.Recoverer)
	mux.Use(metrics.Middleware)

	mux.Get("/api/l1-builder/v1/measurements", srv.appHandler.GetAllowedMeasurements)
	mux.Get("/api/internal/l1-builder/v1/builders", srv.appHandler.GetActiveBuildersNoAuth)

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
	go func() {
		srv.log.Info("Starting internal HTTP server", "listenAddress", srv.cfg.InternalAddr)
		if err := srv.internalSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srv.log.Error("Internal HTTP server failed", "err", err)
		}
	}()
	go func() {
		srv.log.Info("Starting admin HTTP server", "listenAddress", srv.cfg.AdminAddr)
		if err := srv.adminSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srv.log.Error("Admin HTTP server failed", "err", err)
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

	if err := srv.internalSrv.Shutdown(ctx); err != nil {
		srv.log.Error("Graceful HTTP server shutdown failed", "err", err)
	} else {
		srv.log.Info("HTTP server gracefully stopped")
	}

	if err := srv.adminSrv.Shutdown(ctx); err != nil {
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
