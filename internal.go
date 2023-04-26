package patchy

import (
	"context"

	"github.com/dchest/uniuri"
	"github.com/gopatchy/jsrest"
	"github.com/gopatchy/metadata"
	"github.com/gopatchy/path"
)

type getStreamInt struct {
	ch <-chan any

	api    *API
	cfg    *config
	id     string
	sbChan <-chan any
}

type listStreamInt struct {
	ch <-chan []any

	api    *API
	cfg    *config
	sbChan <-chan []any
}

func (api *API) createInt(ctx context.Context, cfg *config, obj any) (any, error) {
	md := metadata.GetMetadata(obj)

	if ctx.Value(ContextWriteID) == nil || md.ID == "" {
		md.ID = uniuri.New()
	}

	if ctx.Value(ContextWriteGeneration) == nil || md.Generation == 0 {
		md.Generation = 1
	}

	obj, err := cfg.checkWrite(ctx, obj, nil, api)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrForbidden, "write check failed (%w)", err)
	}

	err = api.sb.Write(ctx, cfg.apiName, obj)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "write failed (%w)", err)
	}

	obj, err = cfg.checkRead(ctx, obj, api)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrForbidden, "read check failed (%w)", err)
	}

	return obj, nil
}

func (api *API) deleteInt(ctx context.Context, cfg *config, id string, opts *UpdateOpts) error {
	if opts == nil {
		opts = &UpdateOpts{}
	}

	obj, err := api.sb.Read(ctx, cfg.apiName, id, cfg.factory)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "read failed: %s (%w)", id, err)
	}

	if obj == nil {
		return jsrest.Errorf(jsrest.ErrNotFound, "%s", id)
	}

	err = opts.ifMatch(obj)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "match failed (%w)", err)
	}

	_, err = cfg.checkWrite(ctx, nil, obj, api)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrForbidden, "write check failed (%w)", err)
	}

	err = api.sb.Delete(ctx, cfg.apiName, id)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "delete failed: %s (%w)", id, err)
	}

	return nil
}

func (api *API) getInt(ctx context.Context, cfg *config, id string) (any, error) {
	obj, err := api.sb.Read(ctx, cfg.apiName, id, cfg.factory)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "read failed: %s (%w)", id, err)
	}

	if obj == nil {
		return nil, nil
	}

	obj, err = cfg.checkRead(ctx, obj, api)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrForbidden, "read check failed (%w)", err)
	}

	return obj, nil
}

func (api *API) listInt(ctx context.Context, cfg *config, opts *ListOpts) ([]any, error) {
	// TODO: Add query condition pushdown
	if opts == nil {
		opts = &ListOpts{}
	}

	if cfg.listHook != nil {
		err := cfg.listHook(ctx, opts, api)
		if err != nil {
			return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "list hook failed (%w)", err)
		}
	}

	list, err := api.sb.List(ctx, cfg.apiName, cfg.factory)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "read list failed (%w)", err)
	}

	list, err = api.filterList(ctx, cfg, opts, list)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "filter list failed (%w)", err)
	}

	return list, nil
}

func (api *API) replaceInt(ctx context.Context, cfg *config, id string, replace any, opts *UpdateOpts) (any, error) {
	if opts == nil {
		opts = &UpdateOpts{}
	}

	cfg.lock(id)
	defer cfg.unlock(id)

	obj, err := api.sb.Read(ctx, cfg.apiName, id, cfg.factory)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "read failed: %s (%w)", id, err)
	}

	if obj == nil {
		return nil, jsrest.Errorf(jsrest.ErrNotFound, "%s", id)
	}

	err = opts.ifMatch(obj)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "match failed (%w)", err)
	}

	prev, err := cfg.clone(obj)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "clone failed (%w)", err)
	}

	// Metadata is immutable or server-owned
	metadata.ClearMetadata(replace)
	objMD := metadata.GetMetadata(obj)
	replaceMD := metadata.GetMetadata(replace)
	replaceMD.ID = id
	replaceMD.Generation = objMD.Generation + 1

	replace, err = cfg.checkWrite(ctx, replace, prev, api)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrForbidden, "write check failed (%w)", err)
	}

	err = api.sb.Write(ctx, cfg.apiName, replace)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "write failed: %s (%w)", id, err)
	}

	replace, err = cfg.checkRead(ctx, replace, api)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrForbidden, "read check failed (%w)", err)
	}

	return replace, nil
}

func (api *API) updateInt(ctx context.Context, cfg *config, id string, patch map[string]any, opts *UpdateOpts) (any, error) {
	if opts == nil {
		opts = &UpdateOpts{}
	}

	cfg.lock(id)
	defer cfg.unlock(id)

	// Metadata is immutable or server-owned
	delete(patch, "id")
	delete(patch, "etag")
	delete(patch, "generation")

	obj, err := api.sb.Read(ctx, cfg.apiName, id, cfg.factory)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "read failed: %s (%w)", id, err)
	}

	if obj == nil {
		return nil, jsrest.Errorf(jsrest.ErrNotFound, "%s", id)
	}

	err = opts.ifMatch(obj)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "match failed (%w)", err)
	}

	prev, err := cfg.clone(obj)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "clone failed (%w)", err)
	}

	err = path.MergeMap(obj, patch)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrBadRequest, "merge failed (%w)", err)
	}

	metadata.GetMetadata(obj).Generation++

	obj, err = cfg.checkWrite(ctx, obj, prev, api)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrForbidden, "write check failed (%w)", err)
	}

	err = api.sb.Write(ctx, cfg.apiName, obj)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "write failed: %s (%w)", id, err)
	}

	obj, err = cfg.checkRead(ctx, obj, api)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrForbidden, "read check failed (%w)", err)
	}

	return obj, nil
}

func (api *API) streamGetInt(ctx context.Context, cfg *config, id string) (*getStreamInt, error) {
	in, err := api.sb.ReadStream(ctx, cfg.apiName, id, cfg.factory)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "read failed: %s (%w)", id, err)
	}

	out := make(chan any, 100)

	go func() {
		defer close(out)

		for obj := range in {
			obj, err = cfg.checkRead(ctx, obj, api)
			if err != nil {
				break
			}

			out <- obj
		}
	}()

	return &getStreamInt{
		ch:     out,
		api:    api,
		cfg:    cfg,
		id:     id,
		sbChan: in,
	}, nil
}

func (api *API) streamListInt(ctx context.Context, cfg *config, opts *ListOpts) (*listStreamInt, error) {
	if opts == nil {
		opts = &ListOpts{}
	}

	in, err := api.sb.ListStream(ctx, cfg.apiName, cfg.factory)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "read list failed (%w)", err)
	}

	out := make(chan []any, 100)

	go func() {
		defer close(out)

		for list := range in {
			list, err = api.filterList(ctx, cfg, opts, list)
			if err != nil {
				break
			}

			out <- list
		}
	}()

	return &listStreamInt{
		ch:     out,
		api:    api,
		cfg:    cfg,
		sbChan: in,
	}, nil
}

func (gsi *getStreamInt) Close() {
	gsi.api.sb.CloseReadStream(gsi.cfg.apiName, gsi.id, gsi.sbChan)
}

func (gsi *getStreamInt) Chan() <-chan any {
	return gsi.ch
}

func (lsi *listStreamInt) Close() {
	lsi.api.sb.CloseListStream(lsi.cfg.apiName, lsi.sbChan)
}

func (lsi *listStreamInt) Chan() <-chan []any {
	return lsi.ch
}
