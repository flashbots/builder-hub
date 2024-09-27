package httpserver

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flashbots/builder-hub/common"
	"github.com/go-chi/httplog/v2"
	"github.com/stretchr/testify/require"
)

var testServerConfig = &HTTPServerConfig{
	Log: getTestLogger(),
}

func getTestLogger() *httplog.Logger {
	return common.SetupLogger(&common.LoggingOpts{
		Debug:   true,
		JSON:    false,
		Service: "test",
	})
}

func Test_Handlers_Healthcheck_Drain_Undrain(t *testing.T) {
	const (
		latency    = 200 * time.Millisecond
		listenAddr = ":8080"
	)

	//nolint: exhaustruct
	s, err := NewHTTPServer(&HTTPServerConfig{
		DrainDuration: latency,
		ListenAddr:    listenAddr,
		Log:           getTestLogger(),
	})
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

func Test_Handlers_BuilderConfigHub(t *testing.T) {
	routes := []struct {
		method  string
		path    string
		payload []byte
	}{
		// BuilderConfigHub API: https://www.notion.so/flashbots/BuilderConfigHub-1076b4a0d8768074bcdcd1c06c26ec87?pvs=4#10a6b4a0d87680fd81e0cad9bac3b8c5
		{http.MethodGet, "/api/l1-builder/v1/measurements", nil},
		{http.MethodGet, "/api/l1-builder/v1/configuration", nil},
		{http.MethodGet, "/api/l1-builder/v1/builders", nil},
		{http.MethodPost, "/api/l1-builder/v1/register_credentials/rbuilder", []byte(`{"var1":"foo"}`)},
	}

	srv, err := NewHTTPServer(testServerConfig)
	require.NoError(t, err)

	for _, r := range routes {
		var payload io.Reader
		if r.payload != nil {
			payload = bytes.NewReader(r.payload)
		}
		req, err := http.NewRequest(r.method, r.path, payload)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		srv.getRouter().ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	}
}
