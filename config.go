package patchy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/gopatchy/jsrest"
	"github.com/gopatchy/metadata"
)

var ErrMissingAuthCheck = errors.New("missing auth check")

type ListHook func(context.Context, *ListOpts, *API) error

type config struct {
	apiName string
	typeOf  reflect.Type

	factory func() any

	mayRead  func(context.Context, any, *API) error
	mayWrite func(context.Context, any, any, *API) error
	listHook ListHook

	// Per-key read/modify/write (update and replace) operation locking
	// This ensures monotonic generation numbers
	mu    sync.Mutex
	locks map[string]*lock
}

type lock struct {
	mu  sync.Mutex
	ref int
}

type mayRead interface {
	MayRead(context.Context, *API) error
}

type mayWrite[T any] interface {
	MayWrite(context.Context, *T, *API) error
}

func newConfig[T any](apiName string) *config {
	cfg := &config{
		apiName: apiName,
		typeOf:  reflect.TypeOf(new(T)).Elem(),
		factory: func() any { return new(T) },
		locks:   map[string]*lock{},
	}

	typ := cfg.factory()

	if !metadata.HasMetadata(typ) {
		panic("struct missing patchy.Metadata field")
	}

	if _, has := typ.(mayRead); has {
		cfg.mayRead = func(ctx context.Context, obj any, api *API) error {
			obj = convert[T](obj)
			return obj.(mayRead).MayRead(ctx, api)
		}
	}

	if _, has := typ.(mayWrite[T]); has {
		cfg.mayWrite = func(ctx context.Context, obj any, prev any, api *API) error {
			obj = convert[T](obj)
			return obj.(mayWrite[T]).MayWrite(ctx, convert[T](prev), api)
		}
	}

	return cfg
}

func (cfg *config) isSafe() error {
	if cfg.mayRead == nil {
		return fmt.Errorf("%s: MayRead (%w)", cfg.apiName, ErrMissingAuthCheck)
	}

	if cfg.mayWrite == nil {
		return fmt.Errorf("%s: MayWrite (%w)", cfg.apiName, ErrMissingAuthCheck)
	}

	return nil
}

func (cfg *config) checkRead(ctx context.Context, obj any, api *API) (any, error) {
	ret, err := cfg.clone(obj)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "clone failed (%w)", err)
	}

	if cfg.mayRead != nil {
		err := cfg.mayRead(ctx, ret, api)
		if err != nil {
			return nil, jsrest.Errorf(jsrest.ErrUnauthorized, "not authorized to read (%w)", err)
		}
	}

	return ret, nil
}

func (cfg *config) checkReadList(ctx context.Context, list []any, api *API) ([]any, error) { //nolint:unparam
	ret := []any{}

	for _, obj := range list {
		obj, err := cfg.checkRead(ctx, obj, api)
		if err != nil {
			continue
		}

		ret = append(ret, obj)
	}

	return ret, nil
}

func (cfg *config) checkWrite(ctx context.Context, obj, prev any, api *API) (any, error) {
	var ret any

	if obj != nil {
		var err error

		ret, err = cfg.clone(obj)
		if err != nil {
			return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "clone failed (%w)", err)
		}
	}

	if cfg.mayWrite != nil {
		err := cfg.mayWrite(ctx, ret, prev, api)
		if err != nil {
			return nil, jsrest.Errorf(jsrest.ErrUnauthorized, "not authorized to write (%w)", err)
		}
	}

	return ret, nil
}

func (cfg *config) clone(src any) (any, error) {
	js, err := json.Marshal(src)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "JSON marshal (%w)", err)
	}

	dst := cfg.factory()

	err = json.Unmarshal(js, dst)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "JSON unmarhsal (%w)", err)
	}

	return dst, nil
}

func (cfg *config) lock(id string) {
	cfg.mu.Lock()

	entry := cfg.locks[id]
	if entry == nil {
		entry = &lock{}
		cfg.locks[id] = entry
	}
	entry.ref++

	cfg.mu.Unlock()

	entry.mu.Lock()
}

func (cfg *config) unlock(id string) {
	cfg.mu.Lock()

	entry := cfg.locks[id]

	entry.ref--
	if entry.ref == 0 {
		delete(cfg.locks, id)
	}

	cfg.mu.Unlock()

	entry.mu.Unlock()
}

func SetListHookName[T any](api *API, name string, hook ListHook) {
	cfg := api.registry[name]
	if cfg == nil {
		panic(name)
	}

	cfg.listHook = hook
}

func SetListHook[T any](api *API, hook ListHook) {
	SetListHookName[T](api, objName(new(T)), hook)
}

func convert[T any](obj any) *T {
	// Like cast but supports untyped nil
	if obj == nil {
		return nil
	}

	return obj.(*T)
}
