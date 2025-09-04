package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flashbots/builder-hub/adapters/database"
	"github.com/flashbots/builder-hub/adapters/secrets"
	"github.com/flashbots/builder-hub/application"
	"github.com/flashbots/builder-hub/common"
	"github.com/flashbots/builder-hub/domain"
	"github.com/flashbots/builder-hub/httpserver"
	"github.com/flashbots/builder-hub/ports"
	"github.com/google/uuid"
	"github.com/urfave/cli/v2" // imports as package "cli"
)

var flags = []cli.Flag{
	&cli.StringFlag{
		Name:    "listen-addr",
		Value:   "127.0.0.1:8080",
		Usage:   "address to serve API",
		EnvVars: []string{"LISTEN_ADDR"},
	},
	&cli.StringFlag{
		Name:    "admin-addr",
		Value:   "127.0.0.1:8081",
		Usage:   "address to serve admin API",
		EnvVars: []string{"ADMIN_ADDR"},
	},
	&cli.StringFlag{
		Name:    "internal-addr",
		Value:   "127.0.0.1:8082",
		Usage:   "address to serve internal API",
		EnvVars: []string{"INTERNAL_ADDR"},
	},
	&cli.StringFlag{
		Name:    "metrics-addr",
		Value:   "127.0.0.1:8090",
		Usage:   "address to serve Prometheus metrics",
		EnvVars: []string{"METRICS_ADDR"},
	},
	&cli.BoolFlag{
		Name:    "log-json",
		Value:   false,
		Usage:   "log in JSON format",
		EnvVars: []string{"LOG_JSON"},
	},
	&cli.BoolFlag{
		Name:    "log-debug",
		Value:   false,
		Usage:   "log debug messages",
		EnvVars: []string{"LOG_DEBUG"},
	},
	&cli.BoolFlag{
		Name:    "log-uid",
		Value:   false,
		Usage:   "generate a uuid and add to all log messages",
		EnvVars: []string{"LOG_UID"},
	},
	&cli.StringFlag{
		Name:    "log-service",
		Value:   "httpserver",
		Usage:   "add 'service' tag to logs",
		EnvVars: []string{"LOG_SERVICE"},
	},
	&cli.BoolFlag{
		Name:    "pprof",
		Value:   false,
		Usage:   "enable pprof debug endpoint",
		EnvVars: []string{"PPROF"},
	},
	&cli.StringFlag{
		Name:    "admin-basic-user",
		Value:   "admin",
		Usage:   "username for admin Basic Auth",
		EnvVars: []string{"ADMIN_BASIC_USER"},
	},
	&cli.StringFlag{
		Name:    "admin-basic-password-bcrypt",
		Value:   "",
		Usage:   "bcrypt hash of admin password (required to enable admin API, generate with `htpasswd -nbBC 12 admin 'secret' | cut -d: -f2`)",
		EnvVars: []string{"ADMIN_BASIC_PASSWORD_BCRYPT"},
	},
	&cli.BoolFlag{
		Name:    "disable-admin-auth",
		Usage:   "disable admin Basic Auth (local development only)",
		EnvVars: []string{"DISABLE_ADMIN_AUTH"},
	},
	&cli.Int64Flag{
		Name:  "drain-seconds",
		Value: 15,
		Usage: "seconds to wait in drain HTTP request",
	},
	&cli.StringFlag{
		Name:    "postgres-dsn",
		Value:   "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable",
		Usage:   "Postgres DSN",
		EnvVars: []string{"POSTGRES_DSN"},
	},
	&cli.StringFlag{
		Name:    "secret-prefix",
		Value:   "",
		Usage:   "AWS Secret name",
		EnvVars: []string{"AWS_BUILDER_CONFIGS_SECRET_NAME", "AWS_BUILDER_CONFIGS_SECRET_PREFIX"},
	},
	&cli.BoolFlag{
		Name:    "mock-secrets",
		Value:   false,
		Usage:   "Use inmemory secrets service for testing",
		EnvVars: []string{"MOCK_SECRETS"},
	},
}

func main() {
	app := &cli.App{
		Name:    "httpserver",
		Usage:   "Serve API, and metrics",
		Flags:   flags,
		Action:  runCli,
		Version: common.Version,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func runCli(cCtx *cli.Context) error {
	listenAddr := cCtx.String("listen-addr")
	adminAddr := cCtx.String("admin-addr")
	internalAddr := cCtx.String("internal-addr")
	metricsAddr := cCtx.String("metrics-addr")
	logJSON := cCtx.Bool("log-json")
	logDebug := cCtx.Bool("log-debug")
	logUID := cCtx.Bool("log-uid")
	logService := cCtx.String("log-service")
	enablePprof := cCtx.Bool("pprof")
	drainDuration := time.Duration(cCtx.Int64("drain-seconds")) * time.Second
	mockSecretsStorage := cCtx.Bool("mock-secrets")
	adminBasicUser := cCtx.String("admin-basic-user")
	adminPasswordBcrypt := cCtx.String("admin-basic-password-bcrypt")
	disableAdminAuth := cCtx.Bool("disable-admin-auth")

	logTags := map[string]string{
		"version": common.Version,
	}
	if logUID {
		logTags["uid"] = uuid.Must(uuid.NewRandom()).String()
	}

	log := common.SetupLogger(&common.LoggingOpts{
		Service:        logService,
		JSON:           logJSON,
		Debug:          logDebug,
		Concise:        true,
		RequestHeaders: true,
		Tags:           logTags,
	})

	log.With("version", common.Version).Info("starting builder-hub")
	if disableAdminAuth {
		log.Warn("ADMIN AUTH DISABLED! DO NOT USE IN PRODUCTION", "flag", "--disable-admin-auth")
	}

	db, err := database.NewDatabaseService(cCtx.String("postgres-dsn"))
	if err != nil {
		log.Error("failed to create database", "err", err)
		return err
	}
	defer db.Close()

	var sm ports.AdminSecretService

	if mockSecretsStorage {
		log.Info("using mock secrets storage")
		sm = domain.NewMockSecretService()
	} else {
		sm, err = secrets.NewService(cCtx.String("secret-prefix"))
		if err != nil {
			log.Error("failed to create secrets manager", "err", err)
			return err
		}
	}

	builderHub := application.NewBuilderHub(db, sm)
	builderHandler := ports.NewBuilderHubHandler(builderHub, log)

	adminHandler := ports.NewAdminHandler(db, sm, log)
	cfg := &httpserver.HTTPServerConfig{
		ListenAddr:   listenAddr,
		MetricsAddr:  metricsAddr,
		AdminAddr:    adminAddr,
		InternalAddr: internalAddr,
		Log:          log,
		EnablePprof:  enablePprof,

		AdminBasicUser:      adminBasicUser,
		AdminPasswordBcrypt: adminPasswordBcrypt,
		AdminAuthDisabled:   disableAdminAuth,

		DrainDuration:            drainDuration,
		GracefulShutdownDuration: 30 * time.Second,
		ReadTimeout:              60 * time.Second,
		WriteTimeout:             30 * time.Second,
	}

	srv, err := httpserver.NewHTTPServer(cfg, builderHandler, adminHandler)
	if err != nil {
		cfg.Log.Error("failed to create server", "err", err)
		return err
	}

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)
	srv.RunInBackground()
	<-exit

	// Shutdown server once termination signal is received
	srv.Shutdown()
	return nil
}
