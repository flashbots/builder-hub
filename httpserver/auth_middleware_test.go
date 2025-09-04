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

func protectedHandlerDisabled(t *testing.T, disabled bool) http.Handler {
    t.Helper()
    srv := &Server{cfg: &HTTPServerConfig{AdminAuthDisabled: disabled}, log: testLogger()}
    next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = fmt.Fprint(w, "ok")
    })
    return srv.basicAuthMiddleware()(next)
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

func Test_AdminAuth_Disabled_AllowsWithoutCreds(t *testing.T) {
    h := protectedHandlerDisabled(t, true)
    req := httptest.NewRequest(http.MethodGet, "/", nil)
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)
    require.Equal(t, http.StatusOK, rr.Code)

    // When not disabled, should be unauthorized without hash/creds
    h2 := protectedHandlerDisabled(t, false)
    rr2 := httptest.NewRecorder()
    h2.ServeHTTP(rr2, req)
    require.Equal(t, http.StatusUnauthorized, rr2.Code)
}

func Test_PasswordHash(t *testing.T) {
	// Given a password string, ensure a bcrypt (htpasswd-style) hash validates.
	password := "secret"
	htpasswdHash := "$2y$10$Q3mgTfng5mWUlEkLaOA4du0mBIKKYblTcWMqbqehsJM96xF8YV4XC" // create with: htpasswd -nbBC 10 "" 'secret' | cut -d: -f2

	// Generate a bcrypt hash (htpasswd uses bcrypt too). Cost can vary; default is fine for test.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)

	hash2, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)
	require.NotEqual(t, hash, hash2, "bcrypt hashes with same input should differ due to random salt")

	// Correct password must match
	err = bcrypt.CompareHashAndPassword(hash, []byte(password))
	require.NoError(t, err)

	// Wrong password must fail
	err = bcrypt.CompareHashAndPassword(hash, []byte("wrong"))
	require.Error(t, err)

	// Hash from htpasswd works
	err = bcrypt.CompareHashAndPassword([]byte(htpasswdHash), []byte(password))
	require.NoError(t, err)
}
