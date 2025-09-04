package httpserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flashbots/builder-hub/common"
	"github.com/go-chi/httplog/v2"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func testLogger() *httplog.Logger {
	return common.SetupLogger(&common.LoggingOpts{Debug: true, JSON: false, Service: "test"})
}

// helper to create a simple handler protected by the basic auth middleware
func protectedHandler(t *testing.T, user, bcryptHash string) (http.Handler, *Server) {
	t.Helper()
	srv := &Server{cfg: &HTTPServerConfig{AdminBasicUser: user, AdminPasswordBcrypt: bcryptHash}, log: testLogger()}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})
	return srv.basicAuthMiddleware()(next), srv
}

func Test_AdminAuth_DeniesWithoutHash(t *testing.T) {
	h, _ := protectedHandler(t, "admin", "")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func Test_AdminAuth_DeniesWrongCreds(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	require.NoError(t, err)
	h, _ := protectedHandler(t, "admin", string(hash))

	// wrong password
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("admin", "bad")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusUnauthorized, rr.Code)

	// wrong user
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.SetBasicAuth("root", "secret")
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	require.Equal(t, http.StatusUnauthorized, rr2.Code)
}

func Test_AdminAuth_AllowsCorrectCreds(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	require.NoError(t, err)
	h, _ := protectedHandler(t, "admin", string(hash))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("admin", "secret")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}
