package api

import (
	"net/http"

	"github.com/gopatchy/jsrest"
)

func (api *API) put(cfg *config, id string, w http.ResponseWriter, r *http.Request) error {
	replace := cfg.factory()
	opts := parseUpdateOpts(r)

	err := jsrest.Read(r, replace)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "read request failed (%w)", err)
	}

	replace, err = api.replaceInt(r.Context(), cfg, id, replace, opts)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "replace failed (%w)", err)
	}

	err = jsrest.Write(w, replace)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "write response failed (%w)", err)
	}

	return nil
}
