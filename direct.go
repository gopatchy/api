package patchy

import (
	"context"
	"io"

	"github.com/gopatchy/jsrest"
	"github.com/gopatchy/path"
)

func CreateName[TOut, TIn any](ctx context.Context, api *API, name string, obj *TIn) (*TOut, error) {
	cfg := api.registry[name]
	if cfg == nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "unknown type: %s", name)
	}

	created, err := api.createInt(ctx, cfg, obj)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "create failed (%w)", err)
	}

	return created.(*TOut), nil
}

func Create[TOut, TIn any](ctx context.Context, api *API, obj *TIn) (*TOut, error) {
	return CreateName[TOut, TIn](ctx, api, objName(new(TOut)), obj)
}

func DeleteName[TOut any](ctx context.Context, api *API, name, id string, opts *UpdateOpts) error {
	cfg := api.registry[name]
	if cfg == nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "unknown type: %s", name)
	}

	err := api.deleteInt(ctx, cfg, id, opts)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "delete failed (%w)", err)
	}

	return nil
}

func Delete[TOut any](ctx context.Context, api *API, id string, opts *UpdateOpts) error {
	return DeleteName[TOut](ctx, api, objName(new(TOut)), id, opts)
}

func FindName[TOut any](ctx context.Context, api *API, name, shortID string) (*TOut, error) {
	listOpts := &ListOpts{
		Filters: []Filter{
			{
				Path:  "id",
				Op:    "hp",
				Value: shortID,
			},
		},
	}

	objs, err := ListName[TOut](ctx, api, name, listOpts)
	if err != nil {
		return nil, err
	}

	if len(objs) == 0 {
		return nil, jsrest.Errorf(jsrest.ErrNotFound, "no object found with short ID: %s", shortID)
	}

	if len(objs) > 1 {
		return nil, jsrest.Errorf(jsrest.ErrBadRequest, "multiple objects found with short ID: %s", shortID)
	}

	return objs[0], nil
}

func Find[TOut any](ctx context.Context, api *API, shortID string) (*TOut, error) {
	return FindName[TOut](ctx, api, objName(new(TOut)), shortID)
}

func GetName[TOut any](ctx context.Context, api *API, name, id string, opts *GetOpts) (*TOut, error) {
	cfg := api.registry[name]
	if cfg == nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "unknown type: %s", name)
	}

	obj, err := api.getInt(ctx, cfg, id)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "get failed (%w)", err)
	}

	return convert[TOut](obj), nil
}

func Get[TOut any](ctx context.Context, api *API, id string, opts *GetOpts) (*TOut, error) {
	return GetName[TOut](ctx, api, objName(new(TOut)), id, opts)
}

func ListName[TOut any](ctx context.Context, api *API, name string, opts *ListOpts) ([]*TOut, error) {
	cfg := api.registry[name]
	if cfg == nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "unknown type: %s", name)
	}

	list, err := api.listInt(ctx, cfg, opts)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "list failed (%w)", err)
	}

	ret := []*TOut{}
	for _, obj := range list {
		ret = append(ret, obj.(*TOut))
	}

	return ret, nil
}

func List[TOut any](ctx context.Context, api *API, opts *ListOpts) ([]*TOut, error) {
	return ListName[TOut](ctx, api, objName(new(TOut)), opts)
}

func ReplaceName[TOut, TIn any](ctx context.Context, api *API, name, id string, obj *TIn, opts *UpdateOpts) (*TOut, error) {
	cfg := api.registry[name]
	if cfg == nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "unknown type: %s", name)
	}

	replaced, err := api.replaceInt(ctx, cfg, id, obj, opts)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "replace failed (%w)", err)
	}

	return replaced.(*TOut), nil
}

func Replace[TOut, TIn any](ctx context.Context, api *API, id string, obj *TIn, opts *UpdateOpts) (*TOut, error) {
	return ReplaceName[TOut, TIn](ctx, api, objName(new(TOut)), id, obj, opts)
}

func UpdateName[TOut, TIn any](ctx context.Context, api *API, name, id string, obj *TIn, opts *UpdateOpts) (*TOut, error) {
	cfg := api.registry[name]
	if cfg == nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "unknown type: %s", name)
	}

	patch, err := path.ToMap(obj)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrBadRequest, "invalid patch content (%w)", err)
	}

	updated, err := api.updateInt(ctx, cfg, id, patch, opts)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "update failed (%w)", err)
	}

	return updated.(*TOut), nil
}

func Update[TOut, TIn any](ctx context.Context, api *API, id string, obj *TIn, opts *UpdateOpts) (*TOut, error) {
	return UpdateName[TOut, TIn](ctx, api, objName(new(TOut)), id, obj, opts)
}

func StreamGetName[T any](ctx context.Context, api *API, name, id string) (*GetStream[T], error) {
	cfg := api.registry[name]
	if cfg == nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "unknown type: %s", name)
	}

	gsi, err := api.streamGetInt(ctx, cfg, id)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "stream get failed (%w)", err)
	}

	stream := &GetStream[T]{
		ch:  make(chan *T, 100),
		gsi: gsi,
	}

	go func() {
		for obj := range gsi.Chan() {
			stream.writeEvent(convert[T](obj))
		}

		stream.writeError(io.EOF)
	}()

	return stream, nil
}

func StreamGet[TOut any](ctx context.Context, api *API, id string) (*GetStream[TOut], error) {
	return StreamGetName[TOut](ctx, api, objName(new(TOut)), id)
}

func StreamListName[TOut any](ctx context.Context, api *API, name string, opts *ListOpts) (*ListStream[TOut], error) {
	cfg := api.registry[name]
	if cfg == nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "unknown type: %s", name)
	}

	lsi, err := api.streamListInt(ctx, cfg, opts)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "stream list failed (%w)", err)
	}

	stream := &ListStream[TOut]{
		ch:  make(chan []*TOut, 100),
		lsi: lsi,
	}

	go func() {
		for list := range lsi.Chan() {
			typeList := []*TOut{}

			for _, obj := range list {
				typeList = append(typeList, convert[TOut](obj))
			}

			stream.writeEvent(typeList)
		}

		stream.writeError(io.EOF)
	}()

	return stream, nil
}

func StreamList[TOut any](ctx context.Context, api *API, opts *ListOpts) (*ListStream[TOut], error) {
	return StreamListName[TOut](ctx, api, objName(new(TOut)), opts)
}
