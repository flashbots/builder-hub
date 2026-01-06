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
	var err error
	if errs != nil {
		err = errs[0]
		h.log.Warn(msg, "err", err)
	} else {
		h.log.Warn(msg)
	}
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(&badRequestParams{
		Message: msg,
		Error:   err.Error(),
	})
}
