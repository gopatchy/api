package patchy

import (
	"context"
	"io"

	"github.com/gopatchy/jsrest"
	"github.com/gopatchy/metadata"
	"github.com/gopatchy/path"
	"github.com/gopatchy/storebus"
)

func CreateName[T any](ctx context.Context, api *API, name string, obj *T) (*T, error) {
	cfg := api.registry[name]
	if cfg == nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "unknown type: %s", name)
	}

	created, err := api.createInt(ctx, cfg, obj)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "create failed (%w)", err)
	}

	return created.(*T), nil
}

func Create[T any](ctx context.Context, api *API, obj *T) (*T, error) {
	return CreateName[T](ctx, api, apiName[T](), obj)
}

func DeleteName[T any](ctx context.Context, api *API, name, id string, opts *UpdateOpts) error {
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

func Delete[T any](ctx context.Context, api *API, id string, opts *UpdateOpts) error {
	return DeleteName[T](ctx, api, apiName[T](), id, opts)
}

func FindName[T any](ctx context.Context, api *API, name, shortID string) (*T, error) {
	listOpts := &ListOpts{
		Filters: []Filter{
			{
				Path:  "id",
				Op:    "hp",
				Value: shortID,
			},
		},
	}

	objs, err := ListName[T](ctx, api, name, listOpts)
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

func Find[T any](ctx context.Context, api *API, shortID string) (*T, error) {
	return FindName[T](ctx, api, apiName[T](), shortID)
}

func GetName[T any](ctx context.Context, api *API, name, id string, opts *GetOpts) (*T, error) {
	cfg := api.registry[name]
	if cfg == nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "unknown type: %s", name)
	}

	obj, err := api.getInt(ctx, cfg, id)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "get failed (%w)", err)
	}

	return convert[T](obj), nil
}

func Get[T any](ctx context.Context, api *API, id string, opts *GetOpts) (*T, error) {
	return GetName[T](ctx, api, apiName[T](), id, opts)
}

func ListName[T any](ctx context.Context, api *API, name string, opts *ListOpts) ([]*T, error) {
	cfg := api.registry[name]
	if cfg == nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "unknown type: %s", name)
	}

	list, err := api.listInt(ctx, cfg, opts)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "list failed (%w)", err)
	}

	ret := []*T{}
	for _, obj := range list {
		ret = append(ret, obj.(*T))
	}

	return ret, nil
}

func List[T any](ctx context.Context, api *API, opts *ListOpts) ([]*T, error) {
	return ListName[T](ctx, api, apiName[T](), opts)
}

func ReplaceName[T any](ctx context.Context, api *API, name, id string, obj *T, opts *UpdateOpts) (*T, error) {
	cfg := api.registry[name]
	if cfg == nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "unknown type: %s", name)
	}

	replaced, err := api.replaceInt(ctx, cfg, id, obj, opts)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "replace failed (%w)", err)
	}

	return replaced.(*T), nil
}

func Replace[T any](ctx context.Context, api *API, id string, obj *T, opts *UpdateOpts) (*T, error) {
	return ReplaceName[T](ctx, api, apiName[T](), id, obj, opts)
}

func UpdateNameMap[T any](ctx context.Context, api *API, name, id string, patch map[string]any, opts *UpdateOpts) (*T, error) {
	cfg := api.registry[name]
	if cfg == nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "unknown type: %s", name)
	}

	updated, err := api.updateInt(ctx, cfg, id, patch, opts)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "update failed (%w)", err)
	}

	return updated.(*T), nil
}

func UpdateName[T any](ctx context.Context, api *API, name, id string, obj *T, opts *UpdateOpts) (*T, error) {
	patch, err := path.ToMap(obj)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrBadRequest, "invalid patch content (%w)", err)
	}

	return UpdateNameMap[T](ctx, api, name, id, patch, opts)
}

func UpdateMap[T any](ctx context.Context, api *API, id string, patch map[string]any, opts *UpdateOpts) (*T, error) {
	return UpdateNameMap[T](ctx, api, apiName[T](), id, patch, opts)
}

func Update[T any](ctx context.Context, api *API, id string, obj *T, opts *UpdateOpts) (*T, error) {
	return UpdateName[T](ctx, api, apiName[T](), id, obj, opts)
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

func StreamGet[T any](ctx context.Context, api *API, id string) (*GetStream[T], error) {
	return StreamGetName[T](ctx, api, apiName[T](), id)
}

func StreamListName[T any](ctx context.Context, api *API, name string, opts *ListOpts) (*ListStream[T], error) {
	cfg := api.registry[name]
	if cfg == nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "unknown type: %s", name)
	}

	lsi, err := api.streamListInt(ctx, cfg, opts)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "stream list failed (%w)", err)
	}

	stream := &ListStream[T]{
		ch:  make(chan []*T, 100),
		lsi: lsi,
	}

	go func() {
		for list := range lsi.Chan() {
			typeList := []*T{}

			for _, obj := range list {
				typeList = append(typeList, convert[T](obj))
			}

			stream.writeEvent(typeList)
		}

		stream.writeError(io.EOF)
	}()

	return stream, nil
}

func StreamList[T any](ctx context.Context, api *API, opts *ListOpts) (*ListStream[T], error) {
	return StreamListName[T](ctx, api, apiName[T](), opts)
}

func ReplicateInName[TIn, TOut any](ctx context.Context, api *API, name string, in <-chan []*TIn, transform func(in *TIn) (*TOut, error)) error {
	// TODO: Don't use ContextInternal for this (perhaps split to ContextReplication)
	ctx = context.WithValue(ctx, ContextInternal, true)
	ctx = context.WithValue(ctx, ContextWriteID, true)
	ctx = context.WithValue(ctx, ContextWriteGeneration, true)

	for {
		inList := <-in
		if inList == nil {
			return nil
		}

		transList := []*TOut{}

		for _, inObj := range inList {
			transObj, err := transform(inObj)
			if err != nil {
				return err
			}

			err = storebus.UpdateHash(transObj)
			if err != nil {
				return err
			}

			transList = append(transList, transObj)
		}

		curList, err := ListName[TOut](ctx, api, name, nil)
		if err != nil {
			return err
		}

		curByID := map[string]*TOut{}

		for _, curObj := range curList {
			curMD := metadata.GetMetadata(curObj)
			curByID[curMD.ID] = curObj
		}

		for _, transObj := range transList {
			transMD := metadata.GetMetadata(transObj)

			curObj := curByID[transMD.ID]

			if curObj == nil {
				_, err = CreateName[TOut](ctx, api, name, transObj)
				if err != nil {
					return err
				}

				continue
			}

			delete(curByID, transMD.ID)

			curMD := metadata.GetMetadata(curObj)

			if transMD.ETag == curMD.ETag {
				continue
			}

			_, err = UpdateName[TOut](ctx, api, name, curMD.ID, transObj, nil)
			if err != nil {
				return err
			}
		}

		for curID := range curByID {
			err = DeleteName[TOut](ctx, api, name, curID, nil)
			if err != nil {
				return err
			}
		}
	}
}

func ReplicateIn[TIn, TOut any](ctx context.Context, api *API, in <-chan []*TIn, transform func(in *TIn) (*TOut, error)) error {
	return ReplicateInName[TIn, TOut](ctx, api, apiName[TOut](), in, transform)
}
