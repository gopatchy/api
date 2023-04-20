package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/gopatchy/header"
	"github.com/gopatchy/jsrest"
	"github.com/gopatchy/path"
)

func authBearer[T any](r *http.Request, api *API, name, pathToken string) (*http.Request, error) {
	scheme, val := header.ParseAuthorization(r)

	if strings.ToLower(scheme) != "bearer" {
		return r, nil
	}

	bearers, err := ListName[T](
		context.WithValue(r.Context(), ContextInternal, true),
		api,
		name,
		&ListOpts{
			Filters: []*Filter{
				{
					Path:  pathToken,
					Op:    "eq",
					Value: val,
				},
			},
		},
	)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "list tokens for auth failed (%w)", err)
	}

	if len(bearers) != 1 {
		return r, jsrest.Errorf(jsrest.ErrUnauthorized, "token not found")
	}

	bearer := bearers[0]

	err = path.Set(bearer, pathToken, "")
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "clear token failed (%w)", err)
	}

	return r.WithContext(context.WithValue(r.Context(), ContextAuthBearer, bearer)), nil
}

func SetAuthBearerName[T any](api *API, name, pathToken string) {
	api.authBearer = func(r *http.Request, a *API) (*http.Request, error) {
		return authBearer[T](r, a, name, pathToken)
	}
}

func SetAuthBearer[T any](api *API, pathToken string) {
	SetAuthBearerName[T](api, objName(new(T)), pathToken)
}
