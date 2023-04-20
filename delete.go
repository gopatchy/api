package api

import (
	"net/http"

	"github.com/gopatchy/jsrest"
)

func (api *API) delete(cfg *config, id string, w http.ResponseWriter, r *http.Request) error {
	opts := parseUpdateOpts(r)

	err := api.deleteInt(r.Context(), cfg, id, opts)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "delete failed (%w)", err)
	}

	w.WriteHeader(http.StatusNoContent)

	return nil
}
