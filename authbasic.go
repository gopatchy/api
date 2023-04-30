package patchy

import (
	"context"
	"net/http"
	"strings"

	"github.com/gopatchy/header"
	"github.com/gopatchy/jsrest"
	"github.com/gopatchy/path"
	"golang.org/x/crypto/bcrypt"
)

func authBasic[T any](_ http.ResponseWriter, r *http.Request, api *API, name, pathUser, pathPass string) (*http.Request, error) {
	scheme, val := header.ParseAuthorization(r)

	if strings.ToLower(scheme) != "basic" {
		return r, nil
	}

	reqUser, reqPass, err := header.ParseBasic(val)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrBadRequest, "Authorization Basic data parsing failed (%w)", err)
	}

	users, err := ListName[T](
		context.WithValue(r.Context(), ContextAuthBasicLookup, true),
		api,
		name,
		&ListOpts{
			Filters: []Filter{
				{
					Path:  pathUser,
					Op:    "eq",
					Value: reqUser,
				},
			},
		},
	)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "list users for auth failed (%w)", err)
	}

	for _, user := range users {
		userPass, err := path.Get(user, pathPass)
		if err != nil {
			return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "get user password hash failed (%w)", err)
		}

		if userPass == nil {
			continue
		}

		var strPass string

		switch v := userPass.(type) {
		case string:
			strPass = v
		case *string:
			strPass = *v
		default:
			return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "user password hash has invalid type %T", v)
		}

		err = bcrypt.CompareHashAndPassword([]byte(strPass), []byte(reqPass))
		if err == nil {
			err = path.Set(user, pathPass, "")
			if err != nil {
				return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "clear user password hash failed (%w)", err)
			}

			return r.WithContext(context.WithValue(r.Context(), ContextAuthBasic, user)), nil
		}
	}

	return nil, jsrest.Errorf(jsrest.ErrUnauthorized, "user not found or password mismatch")
}

func AddAuthBasicName[T any](api *API, name, pathUser, pathPass string) {
	api.AddRequestHook(func(w http.ResponseWriter, r *http.Request, a *API) (*http.Request, error) {
		return authBasic[T](w, r, a, name, pathUser, pathPass)
	})

	api.authBasic = true
}

func AddAuthBasic[T any](api *API, pathUser, pathPass string) {
	AddAuthBasicName[T](api, apiName[T](), pathUser, pathPass)
}
