package patchy

import (
	"net/http"

	"github.com/gopatchy/jsrest"
)

func (api *API) delete(cfg *config, id string, w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	api.SetEventData(ctx,
		"operation", "delete",
		"typeName", cfg.apiName,
		"id", id,
	)

	opts := parseUpdateOpts(r)

	err := api.deleteInt(ctx, cfg, id, opts)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "delete failed (%w)", err)
	}

	w.WriteHeader(http.StatusNoContent)

	return nil
}
