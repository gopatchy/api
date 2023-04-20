package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/gopatchy/api"
	"github.com/gopatchy/patchyc"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type authBasicType struct {
	api.Metadata
	User string `json:"user" patchy:"authBasicUser"`
	Pass string `json:"pass" patchy:"authBasicPass"`
}

func TestBasicAuthSuccess(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	api.Register[authBasicType](ta.api)

	ctx := context.Background()

	hash, err := bcrypt.GenerateFromPassword([]byte("abcd"), bcrypt.DefaultCost)
	require.NoError(t, err)

	_, err = patchyc.Create[authBasicType](ctx, ta.pyc, &authBasicType{
		User: "foo",
		Pass: string(hash),
	})
	require.NoError(t, err)

	validUser := false

	ta.api.SetRequestHook(func(r *http.Request, _ *api.API) (*http.Request, error) {
		basic := r.Context().Value(api.ContextAuthBasic)
		require.NotNil(t, basic)
		require.IsType(t, &authBasicType{}, basic)
		require.Equal(t, "foo", basic.(*authBasicType).User)
		require.Empty(t, basic.(*authBasicType).Pass)
		validUser = true
		return r, nil
	})

	ta.pyc.SetBasicAuth("foo", "abcd")

	list, err := patchyc.List[authBasicType](ctx, ta.pyc, nil)
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Len(t, list, 1)
	require.True(t, validUser)
}

func TestBasicAuthInvalidUser(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	api.Register[authBasicType](ta.api)

	ctx := context.Background()

	hash, err := bcrypt.GenerateFromPassword([]byte("abcd"), bcrypt.DefaultCost)
	require.NoError(t, err)

	_, err = patchyc.Create[authBasicType](ctx, ta.pyc, &authBasicType{
		User: "foo",
		Pass: string(hash),
	})
	require.NoError(t, err)

	ta.api.SetRequestHook(func(r *http.Request, api *api.API) (*http.Request, error) {
		require.Fail(t, "should not reach request hook")
		return r, nil
	})

	ta.pyc.SetBasicAuth("bar", "abcd")

	_, err = patchyc.List[authBasicType](ctx, ta.pyc, nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "user not found")
}

func TestBasicAuthInvalidPassword(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	api.Register[authBasicType](ta.api)

	ctx := context.Background()

	hash, err := bcrypt.GenerateFromPassword([]byte("abcd"), bcrypt.DefaultCost)
	require.NoError(t, err)

	_, err = patchyc.Create[authBasicType](ctx, ta.pyc, &authBasicType{
		User: "foo",
		Pass: string(hash),
	})
	require.NoError(t, err)

	ta.api.SetRequestHook(func(r *http.Request, api *api.API) (*http.Request, error) {
		require.Fail(t, "should not reach request hook")
		return r, nil
	})

	ta.pyc.SetBasicAuth("foo", "bcde")

	_, err = patchyc.List[authBasicType](ctx, ta.pyc, nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "user password mismatch")
}

func TestBasicAuthOptional(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	api.Register[authBasicType](ta.api)

	ctx := context.Background()

	_, err := patchyc.List[authBasicType](ctx, ta.pyc, nil)
	require.NoError(t, err)
}
