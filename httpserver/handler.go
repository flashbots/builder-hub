package httpserver

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func (srv *Server) handleLivenessCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (srv *Server) handleReadinessCheck(w http.ResponseWriter, r *http.Request) {
	if !srv.isReady.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (srv *Server) handleDrain(w http.ResponseWriter, r *http.Request) {
	if wasReady := srv.isReady.Swap(false); !wasReady {
		return
	}
	// l := logutils.ZapFromRequest(r)
	srv.log.Info("Server marked as not ready")
	time.Sleep(srv.cfg.DrainDuration) // Give LB enough time to detect us not ready
}

func (srv *Server) handleUndrain(w http.ResponseWriter, r *http.Request) {
	if wasReady := srv.isReady.Swap(true); wasReady {
		return
	}
	// l := logutils.ZapFromRequest(r)
	srv.log.Info("Server marked as ready")
}

//
// BuilderConfigHub API: https://www.notion.so/flashbots/BuilderConfigHub-1076b4a0d8768074bcdcd1c06c26ec87?pvs=4#10a6b4a0d87680fd81e0cad9bac3b8c5
//

func (srv *Server) handleGetConfiguration(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, srv.mockGetConfigResponse)
	w.WriteHeader(http.StatusOK)
}

func (srv *Server) handleGetBuilders(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, srv.mockGetBuildersResponse)
	w.WriteHeader(http.StatusOK)
}

func (srv *Server) handleGetMeasurements(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, srv.mockGetMeasurementsResponse)
	w.WriteHeader(http.StatusOK)
}

func (srv *Server) handleRegisterCredentials(w http.ResponseWriter, r *http.Request) {
	// get service name from URL
	service := chi.URLParam(r, "service")
	if !allowedServices[service] {
		srv.log.Error("Invalid service name", "service", service)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		srv.log.Error("Failed to read request body", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	srv.log.Info("Register credentials", "service", service, "body", string(body))
	w.WriteHeader(http.StatusOK)
}
