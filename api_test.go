package api_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/dchest/uniuri"
	"github.com/go-resty/resty/v2"
	"github.com/gopatchy/api"
	"github.com/gopatchy/patchyc"
	"github.com/stretchr/testify/require"
)

type testAPI struct {
	baseURL string
	api     *api.API
	rst     *resty.Client
	pyc     *patchyc.Client

	testBegin int
	testEnd   int
	testError int
	testDone  chan string
}

type testType struct {
	api.Metadata
	Text string `json:"text"`
	Num  int64  `json:"num"`
}

type testTypeRequest struct {
	Text *string `json:"text"`
	Num  *int64  `json:"num"`
}

type testType2 struct {
	api.Metadata
	Text string `json:"text"`
}

type testType3 struct {
	api.Metadata
	Text string `json:"text"`
}

type missingMetadata struct {
	Text string `json:"text"`
}

func newTestAPI(t *testing.T) *testAPI {
	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	a, err := api.NewAPI(dbname)
	require.NoError(t, err)

	err = a.ListenSelfCert("[::]:0")
	require.NoError(t, err)

	return newTestAPIInt(t, a, "https")
}

func newTestAPIInsecure(t *testing.T) *testAPI {
	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	a, err := api.NewAPI(dbname)
	require.NoError(t, err)

	err = a.ListenInsecure("[::]:0")
	require.NoError(t, err)

	return newTestAPIInt(t, a, "http")
}

func newTestAPIInt(t *testing.T, a *api.API, scheme string) *testAPI {
	api.Register[testType](a)
	a.SetStripPrefix("/api")

	ret := &testAPI{
		api:      a,
		testDone: make(chan string, 100),
	}

	a.HandlerFunc("GET", "/_logEvent", func(w http.ResponseWriter, r *http.Request) {
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
		_ = a.Serve()
	}()

	ret.baseURL = fmt.Sprintf("%s://[::1]:%d/api/", scheme, a.Addr().Port)

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

func (tt *testType) MayRead(context.Context, *api.API) error {
	return nil
}

func (tt *testType2) MayWrite(context.Context, *testType2, *api.API) error {
	return nil
}

func (tt *testType3) MayRead(context.Context, *api.API) error {
	return nil
}

func (tt *testType3) MayWrite(context.Context, *testType3, *api.API) error {
	return nil
}

func TestRegisterMissingMetadata(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	a, err := api.NewAPI(dbname)
	require.NoError(t, err)

	defer a.Close()

	require.Panics(t, func() {
		api.Register[missingMetadata](a)
	})
}

func TestIsSafeSuccess(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	a, err := api.NewAPI(dbname)
	require.NoError(t, err)

	defer a.Close()

	api.Register[testType3](a)

	require.NoError(t, a.IsSafe())
}

func TestIsSafeWithoutWrite(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	a, err := api.NewAPI(dbname)
	require.NoError(t, err)

	defer a.Close()

	require.NoError(t, a.IsSafe())

	api.Register[testType](a)

	require.ErrorIs(t, a.IsSafe(), api.ErrMissingAuthCheck)
}

func TestIsSafeWithoutRead(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	a, err := api.NewAPI(dbname)
	require.NoError(t, err)

	defer a.Close()

	require.NoError(t, a.IsSafe())

	api.Register[testType2](a)

	require.ErrorIs(t, a.IsSafe(), api.ErrMissingAuthCheck)
}

func TestCheckSafeSuccess(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	a, err := api.NewAPI(dbname)
	require.NoError(t, err)

	defer a.Close()

	api.Register[testType3](a)

	require.NotPanics(t, a.CheckSafe)
}

func TestCheckSafeWithoutWrite(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	a, err := api.NewAPI(dbname)
	require.NoError(t, err)

	defer a.Close()

	require.NotPanics(t, a.CheckSafe)

	api.Register[testType](a)

	require.Panics(t, a.CheckSafe)
}

func TestCheckSafeWithoutRead(t *testing.T) {
	t.Parallel()

	dbname := fmt.Sprintf("file:%s?mode=memory&cache=shared", uniuri.New())

	a, err := api.NewAPI(dbname)
	require.NoError(t, err)

	defer a.Close()

	require.NotPanics(t, a.CheckSafe)

	api.Register[testType2](a)

	require.Panics(t, a.CheckSafe)
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

	ta.api.SetRequestHook(func(*http.Request, *api.API) (*http.Request, error) {
		return nil, fmt.Errorf("test reject") //nolint:goerr113
	})

	created, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.Error(t, err)
	require.Nil(t, created)
}
