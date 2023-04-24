{{- if and .Info .Info.Title -}}
// {{ .Info.Title }} client

{{ end -}}
package {{ if .Form.Has "packageName" -}} {{ .Form.Get "packageName" }} {{- else -}} goclient {{- end }}

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	{{- if .URLPrefix }}
	"net/url"
	{{- end }}
	"strconv"
	"strings"
	"sync"
	"time"

	//
	{{- if .UsesCivil }}
	"cloud.google.com/go/civil"
	{{- end }}
	"github.com/go-resty/resty/v2"
	"github.com/gopatchy/jsrest"
	"github.com/gopatchy/metadata"
	"github.com/gopatchy/path"
	"golang.org/x/exp/slices"
)

{{- range $type := .Types }}

type {{ $type.TypeUpperCamel }} struct {
	{{- if $type.TopLevel }}
	metadata.Metadata
	listETag string
	{{- end }}

	{{- range $field := .Fields }}
	{{ padRight $field.NameUpperCamel $type.FieldNameMaxLen }} {{ padRight $field.GoType $type.FieldGoTypeMaxLen }} `json:"{{ $field.NameLower }},omitempty"`
	{{- end }}
}

{{- end }}

type GetOpts[T any] struct {
	Prev *T
}

type ListOpts[T any] struct {
	Stream  string
	Limit   int64
	Offset  int64
	After   string
	Sorts   []string
	Filters []Filter

	Prev []*T
}

type Filter struct {
	Path  string
	Op    string
	Value string
}

type UpdateOpts[T any] struct {
	Prev *T
}

type streamEvent struct {
	eventType string
	params    map[string]string
	data      []byte
}

type Client struct {
	rst *resty.Client
}

var (
	ErrNotFound            = fmt.Errorf("not found")
	ErrMultipleFound       = fmt.Errorf("multiple found")
	ErrInvalidStreamEvent  = fmt.Errorf("invalid stream event")
	ErrInvalidStreamFormat = fmt.Errorf("invalid stream format")
)

func NewClient(baseURL string) *Client {
	{{- if .URLPrefix }}
	baseURL, err := url.JoinPath(baseURL, "{{ .URLPrefix }}")
	if err != nil {
		panic(err)
	}
	{{- end }}

	rst := resty.New().
		SetBaseURL(baseURL).
		SetHeader("Accept", "application/json").
		SetJSONEscapeHTML(false)

	// TODO: SetTimeout()
	// TODO: SetRetry*() or roll our own
	// TODO: Add Idempotency-Key support

	return &Client{
		rst: rst,
	}
}

func (c *Client) SetTLSClientConfig(cfg *tls.Config) *Client {
	c.rst.SetTLSClientConfig(cfg)
	return c
}

func (c *Client) SetDebug(debug bool) *Client {
	c.rst.SetDebug(debug)
	return c
}

func (c *Client) SetHeader(name, value string) *Client {
	c.rst.SetHeader(name, value)
	return c
}

func (c *Client) ResetAuth() *Client {
	c.rst.Token = ""
	c.rst.UserInfo = nil

	return c
}

{{- if .AuthBasic }}

func (c *Client) SetBasicAuth(user, pass string) *Client {
	c.ResetAuth()
	c.rst.SetBasicAuth(user, pass)

	return c
}
{{- end }}

{{- if .AuthBearer }}

func (c *Client) SetAuthToken(token string) *Client {
	c.ResetAuth()
	c.rst.SetAuthToken(token)

	return c
}
{{- end }}

func (c *Client) DebugInfo(ctx context.Context) (map[string]any, error) {
	return c.fetchMap(ctx, "_debug")
}

func (c *Client) OpenAPI(ctx context.Context) (map[string]any, error) {
	return c.fetchMap(ctx, "_openapi")
}

func (c *Client) GoClient(ctx context.Context) (string, error) {
	return c.fetchString(ctx, "_client.go")
}

func (c *Client) TSClient(ctx context.Context) (string, error) {
	return c.fetchString(ctx, "_client.ts")
}


{{- range $api := .APIs }}

//// {{ $api.NameUpperCamel }}

func (c *Client) Create{{ $api.NameUpperCamel }}(ctx context.Context, obj *{{ $api.TypeUpperCamel }}) (*{{ $api.TypeUpperCamel }}, error) {
	return CreateName[{{ $api.TypeUpperCamel }}](ctx, c, "{{ $api.NameLower }}", obj)
}

func (c *Client) Delete{{ $api.NameUpperCamel }}(ctx context.Context, id string, opts *UpdateOpts[{{ $api.TypeUpperCamel }}]) error {
	return DeleteName[{{ $api.TypeUpperCamel }}](ctx, c, "{{ $api.NameLower }}", id, opts)
}

func (c *Client) Find{{ $api.NameUpperCamel }}(ctx context.Context, shortID string) (*{{ $api.TypeUpperCamel }}, error) {
	return FindName[{{ $api.TypeUpperCamel }}](ctx, c, "{{ $api.NameLower }}", shortID)
}

func (c *Client) Get{{ $api.NameUpperCamel }}(ctx context.Context, id string, opts *GetOpts[{{ $api.TypeUpperCamel }}]) (*{{ $api.TypeUpperCamel }}, error) {
	return GetName[{{ $api.TypeUpperCamel }}](ctx, c, "{{ $api.NameLower }}", id, opts)
}

func (c *Client) List{{ $api.NameUpperCamel }}(ctx context.Context, opts *ListOpts[{{ $api.TypeUpperCamel }}]) ([]*{{ $api.TypeUpperCamel }}, error) {
	return ListName[{{ $api.TypeUpperCamel }}](ctx, c, "{{ $api.NameLower }}", opts)
}

func (c *Client) Replace{{ $api.NameUpperCamel }}(ctx context.Context, id string, obj *{{ $api.TypeUpperCamel }}, opts *UpdateOpts[{{ $api.TypeUpperCamel }}]) (*{{ $api.TypeUpperCamel }}, error) {
	return ReplaceName[{{ $api.TypeUpperCamel }}](ctx, c, "{{ $api.NameLower }}", id, obj, opts)
}

func (c *Client) Update{{ $api.NameUpperCamel }}(ctx context.Context, id string, obj *{{ $api.TypeUpperCamel }}, opts *UpdateOpts[{{ $api.TypeUpperCamel }}]) (*{{ $api.TypeUpperCamel }}, error) {
	return UpdateName[{{ $api.TypeUpperCamel }}](ctx, c, "{{ $api.NameLower }}", id, obj, opts)
}

func (c *Client) StreamGet{{ $api.NameUpperCamel }}(ctx context.Context, id string, opts *GetOpts[{{ $api.TypeUpperCamel }}]) (*GetStream[{{ $api.TypeUpperCamel }}], error) {
	return StreamGetName[{{ $api.TypeUpperCamel }}](ctx, c, "{{ $api.NameLower }}", id, opts)
}

func (c *Client) StreamList{{ $api.NameUpperCamel }}(ctx context.Context, opts *ListOpts[{{ $api.TypeUpperCamel }}]) (*ListStream[{{ $api.TypeUpperCamel }}], error) {
	return StreamListName[{{ $api.TypeUpperCamel }}](ctx, c, "{{ $api.NameLower }}", opts)
}
{{- end }}

//// Generic

func CreateName[T any](ctx context.Context, c *Client, name string, obj *T) (*T, error) {
	created := new(T)

	resp, err := c.rst.R().
		SetContext(ctx).
		SetPathParam("name", name).
		SetBody(obj).
		SetResult(created).
		Post("{name}")
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, jsrest.ReadError(resp.Body())
	}

	return created, nil
}

func DeleteName[T any](ctx context.Context, c *Client, name, id string, opts *UpdateOpts[T]) error {
	r := c.rst.R().
		SetContext(ctx).
		SetPathParam("name", name).
		SetPathParam("id", id)

	applyUpdateOpts(opts, r)

	resp, err := r.Delete("{name}/{id}")
	if err != nil {
		return err
	}

	if resp.IsError() {
		return jsrest.ReadError(resp.Body())
	}

	return nil
}

func FindName[T any](ctx context.Context, c *Client, name, shortID string) (*T, error) {
	listOpts := &ListOpts[T]{
		Filters: []Filter{
			{
				Path:  "id",
				Op:    "hp",
				Value: shortID,
			},
		},
	}

	objs, err := ListName[T](ctx, c, name, listOpts)
	if err != nil {
		return nil, err
	}

	if len(objs) == 0 {
		return nil, fmt.Errorf("%s (%w)", shortID, ErrNotFound)
	}

	if len(objs) > 1 {
		return nil, fmt.Errorf("%s (%w)", shortID, ErrMultipleFound)
	}

	return objs[0], nil
}

func GetName[T any](ctx context.Context, c *Client, name, id string, opts *GetOpts[T]) (*T, error) {
	obj := new(T)

	r := c.rst.R().
		SetContext(ctx).
		SetPathParam("name", name).
		SetPathParam("id", id).
		SetResult(obj)

	applyGetOpts(opts, r)

	resp, err := r.Get("{name}/{id}")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, nil
	}

	if opts != nil && opts.Prev != nil && resp.StatusCode() == http.StatusNotModified {
		return opts.Prev, nil
	}

	if resp.IsError() {
		return nil, jsrest.ReadError(resp.Body())
	}

	return obj, nil
}

func ListName[T any](ctx context.Context, c *Client, name string, opts *ListOpts[T]) ([]*T, error) {
	objs := []*T{}

	r := c.rst.R().
		SetContext(ctx).
		SetPathParam("name", name).
		SetResult(&objs)

	err := applyListOpts(opts, r)
	if err != nil {
		return nil, err
	}

	resp, err := r.Get("{name}")
	if err != nil {
		return nil, err
	}

	if opts != nil && opts.Prev != nil && resp.StatusCode() == http.StatusNotModified {
		return opts.Prev, nil
	}

	if resp.IsError() {
		return nil, jsrest.ReadError(resp.Body())
	}

	if len(objs) > 0 && resp.Header().Get("ETag") != "" {
		err = path.Set(objs[0], "listETag", resp.Header().Get("ETag"))
		if err != nil {
			return nil, err
		}
	}

	return objs, nil
}

func ReplaceName[T any](ctx context.Context, c *Client, name, id string, obj *T, opts *UpdateOpts[T]) (*T, error) {
	replaced := new(T)

	r := c.rst.R().
		SetContext(ctx).
		SetPathParam("name", name).
		SetPathParam("id", id).
		SetBody(obj).
		SetResult(replaced)

	applyUpdateOpts(opts, r)

	resp, err := r.Put("{name}/{id}")
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, jsrest.ReadError(resp.Body())
	}

	return replaced, nil
}

func UpdateName[T any](ctx context.Context, c *Client, name, id string, obj *T, opts *UpdateOpts[T]) (*T, error) {
	updated := new(T)

	r := c.rst.R().
		SetContext(ctx).
		SetPathParam("name", name).
		SetPathParam("id", id).
		SetBody(obj).
		SetResult(updated)

	applyUpdateOpts(opts, r)

	resp, err := r.Patch("{name}/{id}")
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, jsrest.ReadError(resp.Body())
	}

	return updated, nil
}

func StreamGetName[T any](ctx context.Context, c *Client, name, id string, opts *GetOpts[T]) (*GetStream[T], error) {
	r := c.rst.R().
		SetDoNotParseResponse(true).
		SetHeader("Accept", "text/event-stream").
		SetPathParam("name", name).
		SetPathParam("id", id)

	applyGetOpts(opts, r)

	resp, err := r.Get("{name}/{id}")
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, jsrest.ReadError(resp.Body())
	}

	body := resp.RawBody()
	scan := bufio.NewScanner(body)

	stream := &GetStream[T]{
		ch:   make(chan *T, 100),
		body: body,
	}

	go func() {
		for {
			event, err := readEvent(scan)
			if err != nil {
				stream.writeError(err)
				return
			}

			switch event.eventType {
			case "initial":
				fallthrough
			case "update":
				obj := new(T)

				err = event.decode(obj)
				if err != nil {
					stream.writeError(err)
					return
				}

				stream.writeEvent(obj)

			case "notModified":
				if opts != nil && opts.Prev != nil {
					stream.writeEvent(opts.Prev)
				} else {
					stream.writeError(fmt.Errorf("notModified without If-None-Match (%w)", ErrInvalidStreamEvent))
					return
				}

			case "heartbeat":
				stream.writeHeartbeat()
			}
		}
	}()

	return stream, nil
}

func StreamListName[T any](ctx context.Context, c *Client, name string, opts *ListOpts[T]) (*ListStream[T], error) {
	r := c.rst.R().
		SetDoNotParseResponse(true).
		SetHeader("Accept", "text/event-stream").
		SetPathParam("name", name)

	if opts == nil {
		opts = &ListOpts[T]{}
	}

	err := applyListOpts(opts, r)
	if err != nil {
		return nil, err
	}

	resp, err := r.Get("{name}")
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, jsrest.ReadError(resp.Body())
	}

	body := resp.RawBody()
	scan := bufio.NewScanner(body)

	stream := &ListStream[T]{
		ch:   make(chan []*T, 100),
		body: body,
	}

	switch resp.Header().Get("Stream-Format") {
	case "full":
		go streamListFull(scan, stream, opts)

	case "diff":
		go streamListDiff(scan, stream, opts)

	default:
		stream.Close()
		return nil, fmt.Errorf("%s (%w)", resp.Header().Get("Stream-Format"), ErrInvalidStreamFormat)
	}

	return stream, nil
}

// XXX Start here

func streamListFull[T any](scan *bufio.Scanner, stream *ListStream[T], opts *ListOpts[T]) {
	for {
		event, err := readEvent(scan)
		if err != nil {
			stream.writeError(err)
			return
		}

		switch event.eventType {
		case "list":
			list := []*T{}

			err = event.decode(&list)
			if err != nil {
				stream.writeError(err)
				return
			}

			if len(list) > 0 {
				err = path.Set(list[0], "listETag", event.params["id"])
				if err != nil {
					stream.writeError(err)
					return
				}
			}

			stream.writeEvent(list)

		case "notModified":
			if opts != nil && opts.Prev != nil {
				stream.writeEvent(opts.Prev)
			} else {
				stream.writeError(fmt.Errorf("notModified without If-None-Match (%w)", ErrInvalidStreamEvent))
				return
			}

		case "heartbeat":
			stream.writeHeartbeat()
		}
	}
}

func streamListDiff[T any](scan *bufio.Scanner, stream *ListStream[T], opts *ListOpts[T]) {
	list := []*T{}

	add := func(event *streamEvent) error {
		obj := new(T)

		err := event.decode(obj)
		if err != nil {
			return err
		}

		pos, err := strconv.Atoi(event.params["new-position"])
		if err != nil {
			return err
		}

		list = slices.Insert(list, pos, obj)

		return nil
	}

	remove := func(event *streamEvent) error {
		pos, err := strconv.Atoi(event.params["old-position"])
		if err != nil {
			return err
		}

		list = slices.Delete(list, pos, pos+1)

		return nil
	}

	for {
		event, err := readEvent(scan)
		if err != nil {
			stream.writeError(err)
			return
		}

		switch event.eventType {
		case "add":
			err = add(event)
			if err != nil {
				stream.writeError(err)
				return
			}

		case "update":
			err = remove(event)
			if err != nil {
				stream.writeError(err)
				return
			}

			err = add(event)
			if err != nil {
				stream.writeError(err)
				return
			}

		case "remove":
			err = remove(event)
			if err != nil {
				stream.writeError(err)
				return
			}

		case "sync":
			if len(list) > 0 {
				err = path.Set(list[0], "listETag", event.params["id"])
				if err != nil {
					stream.writeError(err)
					return
				}
			}

			stream.writeEvent(list)

		case "notModified":
			list = opts.Prev
			stream.writeEvent(list)

		case "heartbeat":
			stream.writeHeartbeat()
		}
	}
}

type GetStream[T any] struct {
	ch   chan *T
	body io.ReadCloser

	lastEventReceived time.Time
	err               error
	mu                sync.RWMutex
}

func (gs *GetStream[T]) Close() {
	gs.body.Close()
}

func (gs *GetStream[T]) Chan() <-chan *T {
	return gs.ch
}

func (gs *GetStream[T]) Read() *T {
	return <-gs.Chan()
}

func (gs *GetStream[T]) LastEventReceived() time.Time {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	return gs.lastEventReceived
}

func (gs *GetStream[T]) Error() error {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	return gs.err
}

func (gs *GetStream[T]) writeHeartbeat() {
	gs.mu.Lock()
	gs.lastEventReceived = time.Now()
	gs.mu.Unlock()
}

func (gs *GetStream[T]) writeEvent(obj *T) {
	gs.mu.Lock()
	gs.lastEventReceived = time.Now()
	gs.mu.Unlock()

	gs.ch <- obj
}

func (gs *GetStream[T]) writeError(err error) {
	gs.mu.Lock()
	gs.err = err
	gs.mu.Unlock()

	close(gs.ch)
}

type ListStream[T any] struct {
	ch   chan []*T
	body io.ReadCloser

	lastEventReceived time.Time

	err error

	mu sync.RWMutex
}

func (ls *ListStream[T]) Close() {
	ls.body.Close()
}

func (ls *ListStream[T]) Chan() <-chan []*T {
	return ls.ch
}

func (ls *ListStream[T]) Read() []*T {
	return <-ls.Chan()
}

func (ls *ListStream[T]) LastEventReceived() time.Time {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	return ls.lastEventReceived
}

func (ls *ListStream[T]) Error() error {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	return ls.err
}

func (ls *ListStream[T]) writeHeartbeat() {
	ls.mu.Lock()
	ls.lastEventReceived = time.Now()
	ls.mu.Unlock()
}

func (ls *ListStream[T]) writeEvent(list []*T) {
	ls.mu.Lock()
	ls.lastEventReceived = time.Now()
	ls.mu.Unlock()

	ls.ch <- list
}

func (ls *ListStream[T]) writeError(err error) {
	ls.mu.Lock()
	ls.err = err
	ls.mu.Unlock()

	close(ls.ch)
}

func readEvent(scan *bufio.Scanner) (*streamEvent, error) {
	event := &streamEvent{
		params: map[string]string{},
	}
	data := [][]byte{}

	for scan.Scan() {
		line := scan.Text()

		switch {
		case strings.HasPrefix(line, ":"):
			continue

		case strings.HasPrefix(line, "event: "):
			event.eventType = strings.TrimPrefix(line, "event: ")

		case strings.HasPrefix(line, "data: "):
			data = append(data, bytes.TrimPrefix(scan.Bytes(), []byte("data: ")))

		case line == "":
			event.data = bytes.Join(data, []byte("\n"))
			return event, nil

		case strings.Contains(line, ": "):
			parts := strings.SplitN(line, ": ", 2)
			event.params[parts[0]] = parts[1]
		}
	}

	return nil, io.EOF
}

func (event *streamEvent) decode(out any) error {
	return json.Unmarshal(event.data, out)
}

//// Utility generic

func P[T any](v T) *T {
	return &v
}

//// Internal

func (c *Client) fetchMap(ctx context.Context, path string) (map[string]any, error) {
	ret := map[string]any{}

	resp, err := c.rst.R().
		SetContext(ctx).
		SetResult(&ret).
		Get(path)
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, jsrest.ReadError(resp.Body())
	}

	return ret, nil
}

func (c *Client) fetchString(ctx context.Context, path string) (string, error) {
	resp, err := c.rst.R().
		SetContext(ctx).
		Get(path)
	if err != nil {
		return "", err
	}

	if resp.IsError() {
		return "", jsrest.ReadError(resp.Body())
	}

	return resp.String(), nil
}

func applyGetOpts[T any](opts *GetOpts[T], req *resty.Request) {
	if opts == nil {
		return
	}

	if opts.Prev != nil {
		md := metadata.GetMetadata(opts.Prev)
		req.SetHeader("If-None-Match", fmt.Sprintf(`"%s"`, md.ETag))
	}
}

func applyListOpts[T any](opts *ListOpts[T], req *resty.Request) error {
	if opts == nil {
		return nil
	}

	if opts.Prev != nil && len(opts.Prev) > 0 {
		etag, err := path.Get(opts.Prev[0], "listETag")
		if err != nil {
			return err
		}

		req.SetHeader("If-None-Match", etag.(string))
	}

	if opts.Stream != "" {
		req.SetQueryParam("_stream", opts.Stream)
	}

	if opts.Limit != 0 {
		req.SetQueryParam("_limit", fmt.Sprintf("%d", opts.Limit))
	}

	if opts.Offset != 0 {
		req.SetQueryParam("_offset", fmt.Sprintf("%d", opts.Offset))
	}

	if opts.After != "" {
		req.SetQueryParam("_after", opts.After)
	}

	for _, filter := range opts.Filters {
		req.SetQueryParam(fmt.Sprintf("%s[%s]", filter.Path, filter.Op), filter.Value)
	}

	sorts := url.Values{}

	for _, sort := range opts.Sorts {
		sorts.Add("_sort", sort)
	}

	req.SetQueryParamsFromValues(sorts)

	return nil
}

func applyUpdateOpts[T any](opts *UpdateOpts[T], req *resty.Request) {
	if opts == nil {
		return
	}

	if opts.Prev != nil {
		md := metadata.GetMetadata(opts.Prev)
		req.SetHeader("If-Match", fmt.Sprintf(`"%s"`, md.ETag))
	}
}
