package httpserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flashbots/builder-hub/common"
	"github.com/flashbots/builder-hub/ports"
	"github.com/go-chi/httplog/v2"
	"github.com/stretchr/testify/require"
)

const (
	latency      = 200 * time.Millisecond
	listenAddr   = ":8080"
	internalAddr = ":8081"
	adminAddr    = ":8082"
)

var testServerConfig = &HTTPServerConfig{
	Log: getTestLogger(),
}
var _ = testServerConfig

func getTestLogger() *httplog.Logger {
	return common.SetupLogger(&common.LoggingOpts{
		Debug:   true,
		JSON:    false,
		Service: "test",
	})
}

func Test_Handlers_Healthcheck_Drain_Undrain(t *testing.T) {
	//nolint: exhaustruct
	s, err := NewHTTPServer(&HTTPServerConfig{
		DrainDuration: latency,
		ListenAddr:    listenAddr,
		InternalAddr:  internalAddr,
		AdminAddr:     adminAddr,
		Log:           getTestLogger(),
	}, ports.NewBuilderHubHandler(nil, getTestLogger()), ports.NewAdminHandler(nil, nil, getTestLogger()))
	require.NoError(t, err)

	{ // Check health
		req := httptest.NewRequest(http.MethodGet, "http://localhost"+listenAddr+"/readyz", nil) //nolint:goconst,nolintlint
		w := httptest.NewRecorder()
		s.handleReadinessCheck(w, req)
		resp := w.Result()
		defer resp.Body.Close()
		_, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, "Healthcheck must return `Ok` before draining")
	}

	{ // Drain
		req := httptest.NewRequest(http.MethodGet, "http://localhost"+listenAddr+"/drain", nil)
		w := httptest.NewRecorder()
		start := time.Now()
		s.handleDrain(w, req)
		duration := time.Since(start)
		resp := w.Result()
		defer resp.Body.Close()
		_, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, "Must return `Ok` for calls to `/drain`")
		require.GreaterOrEqual(t, duration, latency, "Must wait long enough during draining")
	}

	{ // Check health
		req := httptest.NewRequest(http.MethodGet, "http://localhost"+listenAddr+"/readyz", nil)
		w := httptest.NewRecorder()
		s.handleReadinessCheck(w, req)
		resp := w.Result()
		defer resp.Body.Close()
		_, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode, "Healthcheck must return `Service Unavailable` after draining")
	}

	{ // Undrain
		req := httptest.NewRequest(http.MethodGet, "http://localhost"+listenAddr+"/undrain", nil)
		w := httptest.NewRecorder()
		s.handleUndrain(w, req)
		resp := w.Result()
		defer resp.Body.Close()
		_, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, "Must return `Ok` for calls to `/undrain`")
		time.Sleep(latency)
	}

	{ // Check health
		req := httptest.NewRequest(http.MethodGet, "http://localhost"+listenAddr+"/readyz", nil)
		w := httptest.NewRecorder()
		s.handleReadinessCheck(w, req)
		resp := w.Result()
		defer resp.Body.Close()
		_, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode, "Healthcheck must return `Ok` after undraining")
	}
}
