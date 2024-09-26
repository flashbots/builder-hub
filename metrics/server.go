package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/VictoriaMetrics/metrics"
	chi "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
)

type MetricsServer struct {
	listenAddr string
	log        *httplog.Logger
	srv        *http.Server
}

func NewMetricsServer(listenAddr string, log *httplog.Logger) *MetricsServer {
	server := &MetricsServer{
		listenAddr: listenAddr,
		log:        log,
	}
	return server
}

func (srv *MetricsServer) Start() error {
	srv.srv = &http.Server{
		Addr:              srv.listenAddr,
		Handler:           srv.getRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := srv.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (srv *MetricsServer) getRouter() http.Handler {
	mux := chi.NewRouter()

	if srv.log != nil {
		mux.Use(httplog.RequestLogger(srv.log))
	}

	mux.Use(middleware.Recoverer)
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics.WritePrometheus(w, true)
	})

	return mux
}

func (srv *MetricsServer) Shutdown(ctx context.Context) error {
	return srv.srv.Shutdown(ctx)
}
