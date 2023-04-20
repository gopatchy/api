package patchy_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/dchest/uniuri"
	"github.com/go-resty/resty/v2"
	"github.com/gopatchy/patchy"
	"github.com/gopatchy/patchyc"
	"github.com/stretchr/testify/require"
)

type testAPI struct {
	baseURL string
	api     *patchy.API
	rst     *resty.Client
	pyc     *patchyc.Client

	testBegin int
	testEnd   int
	testError int
	testDone  chan string
}

type testType struct {
	patchy.Metadata
	Text string `json:"text"`
	Num  int64  `json:"num"`
}

type testTypeRequest struct {
	Text *string `json:"text"`
	Num  *int64  `json:"num"`
}

type testType2 struct {
	patchy.Metadata
	Text string `json:"text"`
}

type testType3 struct {
	patchy.Metadata
	Text string `json:"text"`
}

type missingMetadata struct {
	Text string `json:"text"`
}

func newTestAPI(t *testing.T) *testAPI {
	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	api, err := patchy.NewAPI(dbname)
	require.NoError(t, err)

	err = api.ListenSelfCert("[::]:0")
	require.NoError(t, err)

	return newTestAPIInt(t, api, "https")
}

func newTestAPIInsecure(t *testing.T) *testAPI {
	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	api, err := patchy.NewAPI(dbname)
	require.NoError(t, err)

	err = api.ListenInsecure("[::]:0")
	require.NoError(t, err)

	return newTestAPIInt(t, api, "http")
}

func newTestAPIInt(t *testing.T, api *patchy.API, scheme string) *testAPI {
	patchy.Register[testType](api)
	api.SetStripPrefix("/api")

	ret := &testAPI{
		api:      api,
		testDone: make(chan string, 100),
	}

	api.HandlerFunc("GET", "/_logEvent", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		err := r.ParseForm()
		require.NoError(t, err)

		name := r.Form.Get("name")

		switch r.Form.Get("event") {
		case "begin":
			t.Logf("BEGIN [%s]", name)
			ret.testBegin++

		case "end":
			t.Logf("  END [%s]", name)
			ret.testEnd++
			ret.testDone <- name

		case "error":
			t.Errorf("ERROR [%s] %s", name, r.Form.Get("details"))
			ret.testError++
			ret.testDone <- name

		case "log":
			t.Logf("  LOG [%s] %s", name, r.Form.Get("details"))
		}
	})

	go func() {
		_ = api.Serve()
	}()

	ret.baseURL = fmt.Sprintf("%s://[::1]:%d/api/", scheme, api.Addr().Port)

	ret.rst = resty.New().
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}). //nolint:gosec
		SetHeader("Content-Type", "application/json").
		SetBaseURL(ret.baseURL)

	ret.pyc = patchyc.NewClient(ret.baseURL).
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}) //nolint:gosec

	if os.Getenv("PATCHY_DEBUG") != "" {
		ret.rst.SetDebug(true)
		ret.pyc.SetDebug(true)
	}

	return ret
}

func (ta *testAPI) r() *resty.Request {
	return ta.rst.R()
}

func (ta *testAPI) checkTests(t *testing.T) {
	require.Equal(t, ta.testBegin, ta.testEnd)
	require.NotZero(t, ta.testEnd)
	require.Zero(t, ta.testError)
}

func (ta *testAPI) shutdown(t *testing.T) {
	err := ta.api.Shutdown(context.Background())
	require.NoError(t, err)

	ta.api.Close()
}

func (tt *testType) MayRead(context.Context, *patchy.API) error {
	return nil
}

func (tt *testType2) MayWrite(context.Context, *testType2, *patchy.API) error {
	return nil
}

func (tt *testType3) MayRead(context.Context, *patchy.API) error {
	return nil
}

func (tt *testType3) MayWrite(context.Context, *testType3, *patchy.API) error {
	return nil
}

func TestRegisterMissingMetadata(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	api, err := patchy.NewAPI(dbname)
	require.NoError(t, err)

	defer api.Close()

	require.Panics(t, func() {
		patchy.Register[missingMetadata](api)
	})
}

func TestIsSafeSuccess(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	api, err := patchy.NewAPI(dbname)
	require.NoError(t, err)

	defer api.Close()

	patchy.Register[testType3](api)

	require.NoError(t, api.IsSafe())
}

func TestIsSafeWithoutWrite(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	api, err := patchy.NewAPI(dbname)
	require.NoError(t, err)

	defer api.Close()

	require.NoError(t, api.IsSafe())

	patchy.Register[testType](api)

	require.ErrorIs(t, api.IsSafe(), patchy.ErrMissingAuthCheck)
}

func TestIsSafeWithoutRead(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	api, err := patchy.NewAPI(dbname)
	require.NoError(t, err)

	defer api.Close()

	require.NoError(t, api.IsSafe())

	patchy.Register[testType2](api)

	require.ErrorIs(t, api.IsSafe(), patchy.ErrMissingAuthCheck)
}

func TestCheckSafeSuccess(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	api, err := patchy.NewAPI(dbname)
	require.NoError(t, err)

	defer api.Close()

	patchy.Register[testType3](api)

	require.NotPanics(t, api.CheckSafe)
}

func TestCheckSafeWithoutWrite(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	api, err := patchy.NewAPI(dbname)
	require.NoError(t, err)

	defer api.Close()

	require.NotPanics(t, api.CheckSafe)

	patchy.Register[testType](api)

	require.Panics(t, api.CheckSafe)
}

func TestCheckSafeWithoutRead(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	api, err := patchy.NewAPI(dbname)
	require.NoError(t, err)

	defer api.Close()

	require.NotPanics(t, api.CheckSafe)

	patchy.Register[testType2](api)

	require.Panics(t, api.CheckSafe)
}

func TestAcceptJSON(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	get := &testType{}

	resp, err := ta.r().
		SetHeader("Accept", "text/xml, application/json").
		SetResult(get).
		SetPathParam("id", created.ID).
		Get("testtype/{id}")
	require.NoError(t, err)
	require.False(t, resp.IsError())
	require.Equal(t, "application/json", resp.Header().Get("Content-Type"))
	require.Equal(t, "foo", get.Text)
	require.Equal(t, created.ID, get.ID)
}

func TestAcceptEventStream(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	resp, err := ta.r().
		SetDoNotParseResponse(true).
		SetHeader("Accept", "text/event-stream, text/xml").
		SetPathParam("id", created.ID).
		Get("testtype/{id}")
	require.NoError(t, err)
	require.False(t, resp.IsError())
	require.Equal(t, "text/event-stream", resp.Header().Get("Content-Type"))
	resp.RawBody().Close()
}

func TestAcceptFailure(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	resp, err := ta.r().
		SetHeader("Accept", "unsupported").
		SetPathParam("id", created.ID).
		Get("testtype/{id}")
	require.NoError(t, err)
	require.True(t, resp.IsError())
	require.Equal(t, http.StatusNotAcceptable, resp.StatusCode())
}

func TestAcceptListFailure(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	resp, err := ta.r().
		SetHeader("Accept", "unsupported").
		Get("testtype")
	require.NoError(t, err)
	require.True(t, resp.IsError())
	require.Equal(t, http.StatusNotAcceptable, resp.StatusCode())
}

func TestDebug(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	dbg, err := ta.pyc.DebugInfo(ctx)
	require.NoError(t, err)
	require.NotNil(t, dbg)
	require.NotEmpty(t, dbg.Server.Hostname)
}

func TestRequestHookError(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	ta.api.SetRequestHook(func(*http.Request, *patchy.API) (*http.Request, error) {
		return nil, fmt.Errorf("test reject") //nolint:goerr113
	})

	created, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.Error(t, err)
	require.Nil(t, created)
}
