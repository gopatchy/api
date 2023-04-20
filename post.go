package api

import (
	"net/http"

	"github.com/gopatchy/jsrest"
)

func (api *API) post(cfg *config, w http.ResponseWriter, r *http.Request) error {
	obj := cfg.factory()

	err := jsrest.Read(r, obj)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "read request failed (%w)", err)
	}

	created, err := api.createInt(r.Context(), cfg, obj)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "create failed (%w)", err)
	}

	err = jsrest.Write(w, created)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "write response failed (%w)", err)
	}

	return nil
}
