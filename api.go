package patchy

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dchest/uniuri"
	"github.com/gopatchy/jsrest"
	"github.com/gopatchy/metadata"
	"github.com/gopatchy/path"
	"github.com/gopatchy/potency"
	"github.com/gopatchy/selfcert"
	"github.com/gopatchy/storebus"
	"github.com/julienschmidt/httprouter"
	"github.com/vfaronov/httpheader"
)

type API struct {
	router   *httprouter.Router
	sb       *storebus.StoreBus
	potency  *potency.Potency
	registry map[string]*config

	listener net.Listener
	srv      *http.Server

	openAPI      openAPI
	prefix       string
	baseContext  atomic.Value
	requestHooks []RequestHook

	authBasic  bool
	authBearer bool

	contextValues   map[any]any
	contextValuesMu sync.RWMutex

	eventState eventState
}

type (
	RequestHook func(http.ResponseWriter, *http.Request, *API) (*http.Request, error)
	ContextKey  int
	Metadata    = metadata.Metadata
)

var (
	ErrBuildInfoFailed          = errors.New("failed to read build info")
	ErrHeaderValueMissingQuotes = errors.New("header missing quotes")
	ErrUnknownAcceptType        = errors.New("unknown Accept type")
)

const (
	ContextStub ContextKey = iota

	ContextAuthBasicLookup
	ContextAuthBearerLookup
	ContextReplicate

	ContextAuthBearer
	ContextAuthBasic

	ContextWriteID
	ContextWriteGeneration

	ContextSpanID

	ContextEvent
)

func NewAPI(dbname string) (*API, error) {
	router := httprouter.New()
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	sb, err := storebus.NewStoreBus(dbname)
	if err != nil {
		return nil, err
	}

	api := &API{
		router:   router,
		sb:       sb,
		potency:  potency.NewPotency(router),
		registry: map[string]*config{},
		srv: &http.Server{
			ReadHeaderTimeout: 30 * time.Second,
		},
		contextValues: map[any]any{},
	}

	api.SetBaseContext(context.Background())

	api.AddEventHook(EventHookBuildInfo)
	api.AddEventHook(EventHookSpanID)
	api.AddEventHook(EventHookMetrics)
	api.AddEventHook(EventHookRUsage)

	api.srv.Handler = api

	api.srv.BaseContext = func(_ net.Listener) context.Context {
		return api.baseContext.Load().(context.Context)
	}

	api.router.GET(
		"/_debug",
		func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) { api.handleDebug(w, r) },
	)

	api.router.GET(
		"/_openapi",
		func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) { api.handleOpenAPI(w, r) },
	)

	api.router.ServeFiles(
		"/_swaggerui/*filepath",
		http.FS(swaggerUI),
	)

	api.registerTemplates()

	return api, nil
}

func Register[T any](api *API) {
	RegisterName[T](api, apiName[T](), camelName[T]())
}

func RegisterName[T any](api *API, apiName, camelName string) {
	// TODO: Support nested types
	cfg := newConfig[T](apiName, camelName)
	api.registry[cfg.apiName] = cfg
	api.registerHandlers(fmt.Sprintf("/%s", cfg.apiName), cfg)

	authBasicUserPath, ok := path.FindTagValueType(cfg.typeOf, "patchy", "authBasicUser")
	if ok {
		authBasicPassPath, ok := path.FindTagValueType(cfg.typeOf, "patchy", "authBasicPass")
		if !ok {
			panic("patchy:authBasicUser without patchy:authBasicPass")
		}

		AddAuthBasicName[T](api, apiName, authBasicUserPath, authBasicPassPath)
	}

	authBearerTokenPath, ok := path.FindTagValueType(cfg.typeOf, "patchy", "authBearerToken")
	if ok {
		AddAuthBearerName[T](api, apiName, authBearerTokenPath)
	}
}

func (api *API) SetBaseContext(ctx context.Context) {
	// ContextStub exists to force the stored type to context.valueCtx
	api.baseContext.Store(context.WithValue(ctx, ContextStub, true))
}

func (api *API) SetContextValue(key, val any) {
	api.contextValuesMu.Lock()
	defer api.contextValuesMu.Unlock()

	api.contextValues[key] = val
}

func (api *API) SetStripPrefix(prefix string) {
	api.prefix = prefix

	api.AddRequestHook(func(_ http.ResponseWriter, r *http.Request, _ *API) (*http.Request, error) {
		if !strings.HasPrefix(r.URL.Path, prefix) {
			return nil, jsrest.Errorf(jsrest.ErrNotFound, "not found")
		}

		r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)

		return r, nil
	})
}

// TODO: Provide a way for internal auth request hooks to always happen first

func (api *API) AddRequestHook(hook RequestHook) {
	api.requestHooks = append(api.requestHooks, hook)
}

func (api *API) IsSafe() error {
	for _, cfg := range api.registry {
		err := cfg.isSafe()
		if err != nil {
			return err
		}
	}

	return nil
}

func (api *API) CheckSafe() {
	err := api.IsSafe()
	if err != nil {
		panic(err)
	}
}

func (api *API) Handle(method, path string, handler httprouter.Handle) {
	api.router.Handle(method, path, handler)
}

func (api *API) Handler(method, path string, handler http.Handler) {
	api.router.Handler(method, path, handler)
}

func (api *API) HandlerFunc(method, path string, handler http.HandlerFunc) {
	api.router.HandlerFunc(method, path, handler)
}

func (api *API) ServeFiles(path string, fs http.FileSystem) {
	api.router.ServeFiles(path, fs)
}

func (api *API) ListenSelfCert(bind string) error {
	tlsConfig, err := selfcert.NewTLSConfigFromHostPort(bind)
	if err != nil {
		return err
	}

	api.listener, err = tls.Listen("tcp", bind, tlsConfig)
	if err != nil {
		return err
	}

	return nil
}

func (api *API) ListenTLS(bind, certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
		NextProtos:   []string{"h2"},
	}

	api.listener, err = tls.Listen("tcp", bind, cfg)
	if err != nil {
		return err
	}

	return nil
}

func (api *API) ListenInsecure(bind string) error {
	var err error

	api.listener, err = net.Listen("tcp", bind)
	if err != nil {
		return err
	}

	return nil
}

func (api *API) Addr() *net.TCPAddr {
	return api.listener.Addr().(*net.TCPAddr)
}

func (api *API) Serve() error {
	if api.listener == nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "Serve() called before Listen*()")
	}

	err := api.srv.Serve(api.listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (api *API) Shutdown(ctx context.Context) error {
	err := api.srv.Shutdown(ctx)
	if err != nil {
		return err
	}

	api.eventState.Close()
	api.sb.Close()

	return nil
}

func (api *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ctx = context.WithValue(ctx, ContextSpanID, uniuri.New())

	{
		api.contextValuesMu.RLock()

		for k, v := range api.contextValues {
			ctx = context.WithValue(ctx, k, v)
		}

		api.contextValuesMu.RUnlock()
	}

	// TODO: Add lastRequestHost for queries & other
	ev := api.NewEvent(
		"httpProto", r.Proto,
		"requestHost", r.Host,
		"requestMethod", r.Method,
		"requestPath", r.URL.Path,
		"remoteAddr", r.RemoteAddr,
		"responseCode", 200,
	)

	ctx = context.WithValue(ctx, ContextEvent, ev)

	r = r.WithContext(ctx)

	err := api.serveHTTP(w, r)
	if err != nil {
		jsrest.WriteError(w, err)

		hErr := jsrest.GetHTTPError(err)
		if hErr != nil {
			ev.Set("responseCode", hErr.Code)
		}

		ev.Set("responseError", err.Error())
	}

	api.eventState.WriteEvent(ctx, ev)
}

func (api *API) serveHTTP(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "*")
	w.Header().Set("Timing-Allow-Origin", "*")

	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Max-Age", "86400")

		w.WriteHeader(http.StatusNoContent)

		return nil
	}

	err := r.ParseForm()
	if err != nil {
		return jsrest.Errorf(jsrest.ErrUnauthorized, "parse form failed (%w)", err)
	}

	for _, hook := range api.requestHooks {
		var err error

		r, err = hook(w, r, api)
		if err != nil {
			return jsrest.Errorf(jsrest.ErrInternalServerError, "request hook failed (%w)", err)
		}
	}

	api.potency.ServeHTTP(w, r)

	return nil
}

func (api *API) registerHandlers(base string, cfg *config) {
	api.router.GET(
		base,
		func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
			api.wrapError(api.routeListGET, cfg, w, r)
		},
	)

	api.router.POST(
		base,
		func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
			api.wrapError(api.post, cfg, w, r)
		},
	)

	single := fmt.Sprintf("%s/:id", base)

	api.router.PUT(
		single,
		func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
			api.wrapErrorID(api.put, cfg, ps[0].Value, w, r)
		},
	)

	api.router.PATCH(
		single,
		func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
			api.wrapErrorID(api.patch, cfg, ps[0].Value, w, r)
		},
	)

	api.router.DELETE(
		single,
		func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
			api.wrapErrorID(api.delete, cfg, ps[0].Value, w, r)
		},
	)

	api.router.GET(
		single,
		func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
			api.wrapErrorID(api.routeSingleGET, cfg, ps[0].Value, w, r)
		},
	)
}

func (api *API) routeListGET(cfg *config, w http.ResponseWriter, r *http.Request) error {
	ac := httpheader.Accept(r.Header)

	if m := httpheader.MatchAccept(ac, "application/json"); m.Type != "" {
		return api.getList(cfg, w, r)
	}

	if m := httpheader.MatchAccept(ac, "text/event-stream"); m.Type != "" {
		return api.streamList(cfg, w, r)
	}

	return jsrest.Errorf(jsrest.ErrNotAcceptable, "Accept: %s (%w)", r.Header.Get("Accept"), ErrUnknownAcceptType)
}

func (api *API) routeSingleGET(cfg *config, id string, w http.ResponseWriter, r *http.Request) error {
	ac := httpheader.Accept(r.Header)

	if m := httpheader.MatchAccept(ac, "application/json"); m.Type != "" {
		return api.getObject(cfg, id, w, r)
	}

	if m := httpheader.MatchAccept(ac, "text/event-stream"); m.Type != "" {
		return api.streamGet(cfg, id, w, r)
	}

	return jsrest.Errorf(jsrest.ErrNotAcceptable, "Accept: %s (%w)", r.Header.Get("Accept"), ErrUnknownAcceptType)
}

func (api *API) wrapError(cb func(*config, http.ResponseWriter, *http.Request) error, cfg *config, w http.ResponseWriter, r *http.Request) {
	err := cb(cfg, w, r)
	if err != nil {
		jsrest.WriteError(w, err)
	}
}

func (api *API) wrapErrorID(cb func(*config, string, http.ResponseWriter, *http.Request) error, cfg *config, id string, w http.ResponseWriter, r *http.Request) {
	err := cb(cfg, id, w, r)
	if err != nil {
		jsrest.WriteError(w, err)
	}
}

func (api *API) names() []string {
	names := []string{}
	for name := range api.registry {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

func apiName[T any]() string {
	return strings.ToLower(reflect.TypeOf(new(T)).Elem().Name())
}

func camelName[T any]() string {
	return upperFirst(reflect.TypeOf(new(T)).Elem().Name())
}
