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

type {{ $type.TypeUpperCamel }} struct {
	{{- if $type.TopLevel }}
	metadata.Metadata
	{{- end }}

	{{- range $field := .Fields }}
	{{ padRight $field.NameUpperCamel $type.FieldNameMaxLen }} {{ padRight $field.GoType $type.FieldGoTypeMaxLen }} `json:"{{ $field.NameLower }},omitempty"`
	{{- end }}
}

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

{{- range $api := .APIs }}

//// {{ $api.NameUpperCamel }}

func (c *Client) Create{{ $api.NameUpperCamel }}(ctx context.Context, obj *{{ $api.TypeUpperCamel }}) (*{{ $api.TypeUpperCamel }}, error) {
	return CreateName[{{ $api.TypeUpperCamel }}](ctx, c, "{{ $api.NameLower }}", obj)
}

func (c *Client) Delete{{ $api.NameUpperCamel }}(ctx context.Context, id string, opts *UpdateOpts) error {
	return DeleteName[{{ $api.TypeUpperCamel }}](ctx, c, "{{ $api.NameLower }}", id, opts)
}

func (c *Client) Find{{ $api.NameUpperCamel }}(ctx context.Context, shortID string) (*{{ $api.TypeUpperCamel }}, error) {
	return FindName[{{ $api.TypeUpperCamel }}](ctx, c, "{{ $api.NameLower }}", shortID)
}

func (c *Client) Get{{ $api.NameUpperCamel }}(ctx context.Context, id string, opts *GetOpts) (*{{ $api.TypeUpperCamel }}, error) {
	return GetName[{{ $api.TypeUpperCamel }}](ctx, c, "{{ $api.NameLower }}", id, opts)
}

func (c *Client) List{{ $api.NameUpperCamel }}(ctx context.Context, opts *ListOpts) ([]*{{ $api.TypeUpperCamel }}, error) {
	return ListName[{{ $api.TypeUpperCamel }}](ctx, c, "{{ $api.NameLower }}", opts)
}

func (c *Client) Replace{{ $api.NameUpperCamel }}(ctx context.Context, id string, obj *{{ $api.TypeUpperCamel }}, opts *UpdateOpts) (*{{ $api.TypeUpperCamel }}, error) {
	return ReplaceName[{{ $api.TypeUpperCamel }}](ctx, c, "{{ $api.NameLower }}", id, obj, opts)
}

func (c *Client) Update{{ $api.NameUpperCamel }}(ctx context.Context, id string, obj *{{ $api.TypeUpperCamel }}, opts *UpdateOpts) (*{{ $api.TypeUpperCamel }}, error) {
	return UpdateName[{{ $api.TypeUpperCamel }}](ctx, c, "{{ $api.NameLower }}", id, obj, opts)
}

func (c *Client) StreamGet{{ $api.NameUpperCamel }}(ctx context.Context, id string, opts *GetOpts) (*patchyc.GetStream[{{ $api.TypeUpperCamel }}], error) {
	return StreamGetName[{{ $api.TypeUpperCamel }}](ctx, c, "{{ $api.NameLower }}", id, opts)
}

func (c *Client) StreamList{{ $api.NameUpperCamel }}(ctx context.Context, opts *ListOpts) (*patchyc.ListStream[{{ $api.TypeUpperCamel }}], error) {
	return StreamListName[{{ $api.TypeUpperCamel }}](ctx, c, "{{ $api.NameLower }}", opts)
}
{{- end }}

//// Generic

func CreateName[T any](ctx context.Context, c *Client, name string, obj *T) (*T, error) {
	return patchyc.CreateName[T](ctx, c.patchyClient, name, obj)
}

func DeleteName[T any](ctx context.Context, c *Client, name, id string, opts *UpdateOpts) error {
	return patchyc.DeleteName[T](ctx, c.patchyClient, name, id, opts)
}

func FindName[T any](ctx context.Context, c *Client, name, shortID string) (*T, error) {
	return patchyc.FindName[T](ctx, c.patchyClient, name, shortID)
}

func GetName[T any](ctx context.Context, c *Client, name, id string, opts *GetOpts) (*T, error) {
	return patchyc.GetName[T](ctx, c.patchyClient, name, id, opts)
}

func ListName[T any](ctx context.Context, c *Client, name string, opts *ListOpts) ([]*T, error) {
	return patchyc.ListName[T](ctx, c.patchyClient, name, opts)
}

func ReplaceName[T any](ctx context.Context, c *Client, name, id string, obj *T, opts *UpdateOpts) (*T, error) {
	return patchyc.ReplaceName[T](ctx, c.patchyClient, name, id, obj, opts)
}

func UpdateName[T any](ctx context.Context, c *Client, name, id string, obj *T, opts *UpdateOpts) (*T, error) {
	return patchyc.UpdateName[T](ctx, c.patchyClient, name, id, obj, opts)
}

func StreamGetName[T any](ctx context.Context, c *Client, name, id string, opts *GetOpts) (*patchyc.GetStream[T], error) {
	return patchyc.StreamGetName[T](ctx, c.patchyClient, name, id, opts)
}

func StreamListName[T any](ctx context.Context, c *Client, name string, opts *ListOpts) (*patchyc.ListStream[T], error) {
	return patchyc.StreamListName[T](ctx, c.patchyClient, name, opts)
}

//// Utility generic

func P[T any](v T) *T {
	return patchyc.P(v)
}
