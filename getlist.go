package patchy

import (
	"net/http"

	"github.com/gopatchy/jsrest"
	"github.com/vfaronov/httpheader"
)

func (api *API) getList(cfg *config, w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	api.SetEventData(ctx,
		"operation", "list",
		"typeName", cfg.apiName,
		"stream", false,
	)

	opts, err := api.parseListOpts(r)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrBadRequest, "parse list parameters failed (%w)", err)
	}

	list, err := api.listInt(ctx, cfg, opts)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "list failed (%w)", err)
	}

	etag, err := hashList(list)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "hash list failed (%w)", err)
	}

	if httpheader.MatchWeak(opts.IfNoneMatch, httpheader.EntityTag{Opaque: etag}) {
		w.WriteHeader(http.StatusNotModified)
		return nil
	}

	err = jsrest.WriteList(w, list, etag)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "write list failed (%w)", err)
	}

	return nil
}
