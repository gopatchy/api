package patchy

import (
	"fmt"
	"net/http"

	"github.com/gopatchy/jsrest"
	"github.com/gopatchy/metadata"
	"github.com/vfaronov/httpheader"
)

func (api *API) getObject(cfg *config, id string, w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	api.SetEventData(ctx,
		"operation", "get",
		"typeName", cfg.apiName,
		"id", id,
		"stream", false,
	)

	opts := parseGetOpts(r)

	obj, err := api.getInt(ctx, cfg, id)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "get failed (%w)", err)
	}

	if obj == nil {
		return jsrest.Errorf(jsrest.ErrNotFound, "%s", id)
	}

	md := metadata.GetMetadata(obj)
	gen := fmt.Sprintf("generation:%d", md.Generation)

	if httpheader.MatchWeak(opts.IfNoneMatch, httpheader.EntityTag{Opaque: md.ETag}) ||
		httpheader.MatchWeak(opts.IfNoneMatch, httpheader.EntityTag{Opaque: gen}) {
		w.WriteHeader(http.StatusNotModified)
		return nil
	}

	err = jsrest.Write(w, obj)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "write response failed (%w)", err)
	}

	return nil
}
