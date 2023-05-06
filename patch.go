package patchy

import (
	"net/http"

	"github.com/gopatchy/jsrest"
)

func (api *API) patch(cfg *config, id string, w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	api.SetEventData(ctx,
		"operation", "update",
		"typeName", cfg.apiName,
		"id", id,
	)

	patch := map[string]any{}
	opts := parseUpdateOpts(r)

	err := jsrest.Read(r, &patch)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "read request failed (%w)", err)
	}

	obj, err := api.updateInt(ctx, cfg, id, patch, opts)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "update failed (%w)", err)
	}

	err = jsrest.Write(w, obj)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "write response failed (%w)", err)
	}

	return nil
}
