package httpserver

import (
	"net/http"
	"time"
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

func (srv *Server) handleTestPanic(w http.ResponseWriter, r *http.Request) {
	panic("foo")
	// w.WriteHeader(http.StatusOK)
}

//
// BuilderConfigHub API: https://www.notion.so/flashbots/BuilderConfigHub-1076b4a0d8768074bcdcd1c06c26ec87?pvs=4#10a6b4a0d87680fd81e0cad9bac3b8c5
//
