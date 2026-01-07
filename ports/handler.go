package ports

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/httplog/v2"
)

type handler struct {
	log *httplog.Logger
}

type badRequestParams struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

func (h *handler) BadRequest(w http.ResponseWriter, r *http.Request, msg string, errs ...error) {
	w.WriteHeader(http.StatusBadRequest)
	if errs != nil {
		err := errs[0]
		h.log.Warn(msg, "err", err)
		_ = json.NewEncoder(w).Encode(&badRequestParams{
			Message: msg,
			Error:   err.Error(),
		})
		return
	}
	h.log.Warn(msg)
	_ = json.NewEncoder(w).Encode(&badRequestParams{
		Message: msg,
	})
}
