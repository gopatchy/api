//nolint:goerr113
package patchy_test

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/dchest/uniuri"
	"github.com/go-resty/resty/v2"
	"github.com/gopatchy/patchy"
	"github.com/gopatchy/patchyc"
	"github.com/stretchr/testify/require"
)

type testAPI struct {
	baseURL     string
	baseBaseURL string
	api         *patchy.API
	rst         *resty.Client
	pyc         *patchyc.Client

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

type mayType struct {
	patchy.Metadata
	Text1 string
}

type authBearerType struct {
	patchy.Metadata
	Name  string `json:"name"`
	Token string `json:"token" patchy:"authBearerToken"`
}

type authBasicType struct {
	patchy.Metadata
	User string `json:"user" patchy:"authBasicUser"`
	Pass string `json:"pass" patchy:"authBasicPass"`
}

func (mt *mayType) MayRead(ctx context.Context, api *patchy.API) error {
	if ctx.Value(refuseRead) != nil {
		return fmt.Errorf("may not read")
	}

	t1r := ctx.Value(text1Read)
	if t1r != nil {
		mt.Text1 = t1r.(string)
	}

	nt1 := ctx.Value(newText1)
	if nt1 != nil {
		// Use a separate context so we don't recursively create objects
		_, err := patchy.Create[mayType](context.Background(), api, &mayType{Text1: nt1.(string)}) //nolint:contextcheck
		if err != nil {
			return err
		}
	}

	return nil
}

func (mt *mayType) MayWrite(ctx context.Context, prev *mayType, _ *patchy.API) error {
	if ctx.Value(refuseWrite) != nil {
		return fmt.Errorf("may not write")
	}

	t1w := ctx.Value(text1Write)
	if t1w != nil {
		mt.Text1 = t1w.(string)
	}

	return nil
}

type contextKey int

const (
	refuseRead contextKey = iota
	refuseWrite
	text1Read
	text1Write
	newText1
)

func requestHook(r *http.Request, _ *patchy.API) (*http.Request, error) {
	ctx := r.Context()

	if r.Header.Get("X-Refuse-Read") != "" {
		ctx = context.WithValue(ctx, refuseRead, true)
	}

	if r.Header.Get("X-Refuse-Write") != "" {
		ctx = context.WithValue(ctx, refuseWrite, true)
	}

	t1r := r.Header.Get("X-Text1-Read")
	if t1r != "" {
		ctx = context.WithValue(ctx, text1Read, t1r)
	}

	t1w := r.Header.Get("X-Text1-Write")
	if t1w != "" {
		ctx = context.WithValue(ctx, text1Write, t1w)
	}

	nt1 := r.Header.Get("X-NewText1")
	if nt1 != "" {
		ctx = context.WithValue(ctx, newText1, nt1)
	}

	return r.WithContext(ctx), nil
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
	ctx := context.Background()

	patchy.Register[testType](api)
	api.SetStripPrefix("/api")

	ret := &testAPI{
		api:      api,
		testDone: make(chan string, 100),
	}

	api.SetRequestHook(requestHook)
	patchy.Register[mayType](api)

	patchy.Register[authBearerType](api)

	_, err := patchy.Create[authBearerType](ctx, api, &authBearerType{
		Name:  "foo",
		Token: "abcd",
	})
	require.NoError(t, err)

	patchy.Register[authBasicType](api)

	_, err = patchy.Create[authBasicType](ctx, api, &authBasicType{
		User: "foo",
		Pass: "$2a$10$ARCRvjao7aP7CU1Ck8rlqez3FkWwJZY1oe62sxGCA12fxeRcqj0K6", // abcd
	})
	require.NoError(t, err)

	api.HandlerFunc("GET", "/_logEvent", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		err := r.ParseForm()
		require.NoError(t, err)

		name := r.Form.Get("name")

		switch r.Form.Get("event") {
		case "begin":
			t.Logf("[%s] BEGIN", name)
			ret.testBegin++

		case "end":
			t.Logf("[%s] END", name)
			ret.testEnd++
			ret.testDone <- name

		case "error":
			t.Errorf("[%s] ERROR: %s", name, r.Form.Get("details"))
			ret.testError++
			ret.testDone <- name

		case "log":
			t.Logf("[%s] LOG: %s", name, r.Form.Get("details"))
		}
	})

	go func() {
		_ = api.Serve()
	}()

	ret.baseBaseURL = fmt.Sprintf("%s://[::1]:%d/", scheme, api.Addr().Port)
	ret.baseURL = fmt.Sprintf("%sapi/", ret.baseBaseURL)

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

func runNoError(ctx1 context.Context, t *testing.T, dir string, env map[string]string, name string, arg ...string) string {
	ctx2, cancel := context.WithCancel(ctx1)
	t.Cleanup(cancel)

	cmd := exec.CommandContext(ctx2, name, arg...)

	if dir != "" {
		cmd.Dir = dir
	}

	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	t.Logf("[in %s] %s %s", dir, name, strings.Join(arg, " "))

	out, err := cmd.Output()
	stderr := getStderr(err)

	if len(out) > 0 {
		t.Logf("STDOUT:\n%s", string(out))
	}

	if len(stderr) > 0 {
		t.Logf("STDERR:\n%s", stderr)
	}

	if err != nil && strings.Contains(err.Error(), "signal: killed") {
		return string(out)
	}

	require.NoError(t, err)

	return string(out)
}

func getStderr(err error) string {
	ee := &exec.ExitError{}
	if errors.As(err, &ee) {
		return string(ee.Stderr)
	}

	return ""
}
