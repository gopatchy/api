{{- if and .Info .Info.Title -}}
// {{ .Info.Title }} client

{{ end -}}
package {{ if .Form.Has "packageName" -}} {{ .Form.Get "packageName" }} {{- else -}} goclient {{- end }}

import (
	"context"
	"crypto/tls"
	{{- if .URLPrefix }}
	"net/url"
	{{- end }}
	{{- if .UsesTime }}
	"time"
	{{- end }}

	//
	{{- if .UsesCivil }}
	"cloud.google.com/go/civil"
	{{- end }}
	"github.com/gopatchy/metadata"
	"github.com/gopatchy/patchyc"
)

type (
	Filter     = patchyc.Filter
	GetOpts    = patchyc.GetOpts
	ListOpts   = patchyc.ListOpts
	UpdateOpts = patchyc.UpdateOpts
)

{{- range $type := .Types }}
{{- if $type.NameLower }}

// TODO: Less pointers throughout

type {{ $type.TypeUpperCamel }} struct {
	metadata.Metadata
	{{- range $field := .Fields }}
	{{ padRight $field.NameUpperCamel $type.FieldNameMaxLen }} {{ padRight $field.GoType $type.FieldGoTypeMaxLen }} `json:"{{ $field.NameLower }},omitempty"`
	{{- end }}
}

{{- else }}

type {{ $type.TypeUpperCamel }} struct {
	{{- range $field := .Fields }}
	{{ padRight $field.NameUpperCamel $type.FieldNameMaxLen }} {{ padRight $field.GoType $type.FieldGoTypeMaxLen }} `json:"{{ $field.NameLower }},omitempty"`
	{{- end }}
}

{{- end }}
{{- end }}

type Client struct {
	patchyClient *patchyc.Client
}

func NewClient(baseURL string) *Client {
	{{- if .URLPrefix }}
	baseURL, err := url.JoinPath(baseURL, "{{ .URLPrefix }}")
	if err != nil {
		panic(err)
	}
	{{- end }}

	return &Client{
		patchyClient: patchyc.NewClient(baseURL),
	}
}

func (c *Client) SetTLSClientConfig(cfg *tls.Config) *Client {
	c.patchyClient.SetTLSClientConfig(cfg)
	return c
}

func (c *Client) SetDebug(debug bool) *Client {
	c.patchyClient.SetDebug(debug)
	return c
}

func (c *Client) SetHeader(name, value string) *Client {
	c.patchyClient.SetHeader(name, value)
	return c
}

func (c *Client) OpenAPI(ctx context.Context) (map[string]any, error) {
	return c.patchyClient.OpenAPI(ctx)
}

func (c *Client) DebugInfo(ctx context.Context) (map[string]any, error) {
	return c.patchyClient.DebugInfo(ctx)
}

{{- if .AuthBasic }}

func (c *Client) SetBasicAuth(user, pass string) *Client {
	c.patchyClient.SetBasicAuth(user, pass)
	return c
}
{{- end }}

{{- if .AuthBearer }}

func (c *Client) SetAuthToken(token string) *Client {
	c.patchyClient.SetAuthToken(token)
	return c
}
{{- end }}

{{- range $type := .Types }}
{{- if not $type.NameLower }} {{- continue }} {{- end }}

//// {{ $type.NameUpperCamel }}

func (c *Client) Create{{ $type.NameUpperCamel }}(ctx context.Context, obj *{{ $type.TypeUpperCamel }}) (*{{ $type.TypeUpperCamel }}, error) {
	return CreateName[{{ $type.TypeUpperCamel }}, {{ $type.TypeUpperCamel }}](ctx, c, "{{ $type.NameLower }}", obj)
}

func (c *Client) Delete{{ $type.NameUpperCamel }}(ctx context.Context, id string, opts *UpdateOpts) error {
	return DeleteName[{{ $type.TypeUpperCamel }}](ctx, c, "{{ $type.NameLower }}", id, opts)
}

func (c *Client) Find{{ $type.NameUpperCamel }}(ctx context.Context, shortID string) (*{{ $type.TypeUpperCamel }}, error) {
	return FindName[{{ $type.TypeUpperCamel }}](ctx, c, "{{ $type.NameLower }}", shortID)
}

func (c *Client) Get{{ $type.NameUpperCamel }}(ctx context.Context, id string, opts *GetOpts) (*{{ $type.TypeUpperCamel }}, error) {
	return GetName[{{ $type.TypeUpperCamel }}](ctx, c, "{{ $type.NameLower }}", id, opts)
}

func (c *Client) List{{ $type.NameUpperCamel }}(ctx context.Context, opts *ListOpts) ([]*{{ $type.TypeUpperCamel }}, error) {
	return ListName[{{ $type.TypeUpperCamel }}](ctx, c, "{{ $type.NameLower }}", opts)
}

func (c *Client) Replace{{ $type.NameUpperCamel }}(ctx context.Context, id string, obj *{{ $type.TypeUpperCamel }}, opts *UpdateOpts) (*{{ $type.TypeUpperCamel }}, error) {
	return ReplaceName[{{ $type.TypeUpperCamel }}, {{ $type.TypeUpperCamel }}](ctx, c, "{{ $type.NameLower }}", id, obj, opts)
}

func (c *Client) Update{{ $type.NameUpperCamel }}(ctx context.Context, id string, obj *{{ $type.TypeUpperCamel }}, opts *UpdateOpts) (*{{ $type.TypeUpperCamel }}, error) {
	return UpdateName[{{ $type.TypeUpperCamel }}, {{ $type.TypeUpperCamel }}](ctx, c, "{{ $type.NameLower }}", id, obj, opts)
}

func (c *Client) StreamGet{{ $type.NameUpperCamel }}(ctx context.Context, id string, opts *GetOpts) (*patchyc.GetStream[{{ $type.TypeUpperCamel }}], error) {
	return StreamGetName[{{ $type.TypeUpperCamel }}](ctx, c, "{{ $type.NameLower }}", id, opts)
}

func (c *Client) StreamList{{ $type.NameUpperCamel }}(ctx context.Context, opts *ListOpts) (*patchyc.ListStream[{{ $type.TypeUpperCamel }}], error) {
	return StreamListName[{{ $type.TypeUpperCamel }}](ctx, c, "{{ $type.NameLower }}", opts)
}
{{- end }}

//// Generic

func CreateName[TOut, TIn any](ctx context.Context, c *Client, name string, obj *TIn) (*TOut, error) {
	return patchyc.CreateName[TOut, TIn](ctx, c.patchyClient, name, obj)
}

func DeleteName[TOut any](ctx context.Context, c *Client, name, id string, opts *UpdateOpts) error {
	return patchyc.DeleteName[TOut](ctx, c.patchyClient, name, id, opts)
}

func FindName[TOut any](ctx context.Context, c *Client, name, shortID string) (*TOut, error) {
	return patchyc.FindName[TOut](ctx, c.patchyClient, name, shortID)
}

func GetName[TOut any](ctx context.Context, c *Client, name, id string, opts *GetOpts) (*TOut, error) {
	return patchyc.GetName[TOut](ctx, c.patchyClient, name, id, opts)
}

func ListName[TOut any](ctx context.Context, c *Client, name string, opts *ListOpts) ([]*TOut, error) {
	return patchyc.ListName[TOut](ctx, c.patchyClient, name, opts)
}

func ReplaceName[TOut, TIn any](ctx context.Context, c *Client, name, id string, obj *TIn, opts *UpdateOpts) (*TOut, error) {
	return patchyc.ReplaceName[TOut, TIn](ctx, c.patchyClient, name, id, obj, opts)
}

func UpdateName[TOut, TIn any](ctx context.Context, c *Client, name, id string, obj *TIn, opts *UpdateOpts) (*TOut, error) {
	return patchyc.UpdateName[TOut, TIn](ctx, c.patchyClient, name, id, obj, opts)
}

func StreamGetName[TOut any](ctx context.Context, c *Client, name, id string, opts *GetOpts) (*patchyc.GetStream[TOut], error) {
	return patchyc.StreamGetName[TOut](ctx, c.patchyClient, name, id, opts)
}

func StreamListName[TOut any](ctx context.Context, c *Client, name string, opts *ListOpts) (*patchyc.ListStream[TOut], error) {
	return patchyc.StreamListName[TOut](ctx, c.patchyClient, name, opts)
}

//// Utility generic

func P[T any](v T) *T {
	return patchyc.P(v)
}
