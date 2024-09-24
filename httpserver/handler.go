package httpserver

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
// 	m := s.metricsSrv.Float64Histogram(
// 		"request_duration_api",
// 		"API request handling duration",
// 		metrics.UomMicroseconds,
// 		metrics.BucketsRequestDuration...,
// 	)
// 	defer func(start time.Time) {
// 		m.Record(r.Context(), float64(time.Since(start).Microseconds()))
// 	}(time.Now())

// 	// do work

// 	w.WriteHeader(http.StatusOK)
// }

func (s *Server) handleLivenessCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleReadinessCheck(w http.ResponseWriter, r *http.Request) {
	if !s.isReady.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDrain(w http.ResponseWriter, r *http.Request) {
	if wasReady := s.isReady.Swap(false); !wasReady {
		return
	}
	// l := logutils.ZapFromRequest(r)
	s.log.Info("Server marked as not ready")
	time.Sleep(s.cfg.DrainDuration) // Give LB enough time to detect us not ready
}

func (s *Server) handleUndrain(w http.ResponseWriter, r *http.Request) {
	if wasReady := s.isReady.Swap(true); wasReady {
		return
	}
	// l := logutils.ZapFromRequest(r)
	s.log.Info("Server marked as ready")
}

//
// BuilderConfigHub API: https://www.notion.so/flashbots/BuilderConfigHub-1076b4a0d8768074bcdcd1c06c26ec87?pvs=4#10a6b4a0d87680fd81e0cad9bac3b8c5
//

func (s *Server) handleGetConfiguration(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, s.mockGetConfigResponse)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleGetBuilders(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, s.mockGetBuildersResponse)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleGetMeasurements(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, s.mockGetMeasurementsResponse)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleRegisterCredentials(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")

	// read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.log.Error("Failed to read request body", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	s.log.Info("Register credentials", "service", service, "body", string(body))
	w.WriteHeader(http.StatusOK)
}
