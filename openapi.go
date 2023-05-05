package patchy

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/gopatchy/jsrest"
	"github.com/gopatchy/path"
	"golang.org/x/net/idna"
)

type (
	OpenAPI     = openapi3.T
	OpenAPIInfo = openapi3.Info
)

type openAPI struct {
	info *OpenAPIInfo
}

func (api *API) SetOpenAPIInfo(info *OpenAPIInfo) {
	api.openAPI.info = info
}

func (api *API) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	api.AddEventData(ctx, "operation", "openapi")

	err := api.handleOpenAPIInt(w, r)
	if err != nil {
		jsrest.WriteError(w, err)
	}
}

func (api *API) handleOpenAPIInt(w http.ResponseWriter, r *http.Request) error {
	t, err := api.buildOpenAPIGlobal(r)
	if err != nil {
		return err
	}

	for _, name := range api.names() {
		err = api.buildOpenAPIType(t, api.registry[name])
		if err != nil {
			return err
		}
	}

	js, err := t.MarshalJSON()
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "marshal JSON failed (%w)", err)
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(js)

	return nil
}

func (api *API) buildOpenAPIGlobal(r *http.Request) (*openapi3.T, error) {
	baseURL, err := api.requestBaseURL(r)
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "get base URL failed (%w)", err)
	}

	errorSchema, err := generateSchemaRef(reflect.TypeOf(&jsrest.JSONError{}))
	if err != nil {
		return nil, jsrest.Errorf(jsrest.ErrInternalServerError, "generate schema ref failed (%w)", err)
	}

	t := &openapi3.T{
		OpenAPI:  "3.0.3",
		Paths:    openapi3.Paths{},
		Tags:     openapi3.Tags{},
		Security: openapi3.SecurityRequirements{},

		Components: &openapi3.Components{
			RequestBodies:   openapi3.RequestBodies{},
			SecuritySchemes: openapi3.SecuritySchemes{},

			Headers: openapi3.Headers{
				"etag": &openapi3.HeaderRef{
					Value: &openapi3.Header{
						Parameter: openapi3.Parameter{
							Name: "ETag",
							In:   "header",
							Schema: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: "string",
								},
							},
						},
					},
				},

				"idempotency-key": &openapi3.HeaderRef{
					Value: &openapi3.Header{
						Parameter: openapi3.Parameter{
							Name: "Idempotency-Key",
							In:   "header",
							Schema: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: "string",
								},
							},
						},
					},
				},

				"if-match": &openapi3.HeaderRef{
					Value: &openapi3.Header{
						Parameter: openapi3.Parameter{
							Name: "If-Match",
							In:   "header",
							Schema: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: "string",
								},
							},
						},
					},
				},

				"if-none-match": &openapi3.HeaderRef{
					Value: &openapi3.Header{
						Parameter: openapi3.Parameter{
							Name: "If-None-Match",
							In:   "header",
							Schema: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: "string",
								},
							},
						},
					},
				},
			},

			Parameters: openapi3.ParametersMap{
				"id": &openapi3.ParameterRef{
					Value: &openapi3.Parameter{
						Name:        "id",
						In:          "path",
						Description: "Object ID",
						Required:    true,
						Schema: &openapi3.SchemaRef{
							Ref: "#/components/schemas/id",
						},
					},
				},

				"_stream": &openapi3.ParameterRef{
					Value: &openapi3.Parameter{
						Name:        "_stream",
						In:          "query",
						Description: "EventStream (List) format",
						Schema: &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type: "enum",
								Enum: []any{
									"full",
									"diff",
								},
							},
						},
					},
				},

				"_limit": &openapi3.ParameterRef{
					Value: &openapi3.Parameter{
						Name:        "_limit",
						In:          "query",
						Description: "Limit number of objects returned",
						Schema: &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type: "integer",
							},
						},
					},
				},

				"_offset": &openapi3.ParameterRef{
					Value: &openapi3.Parameter{
						Name:        "_offset",
						In:          "query",
						Description: "Skip number of objects at start of list",
						Schema: &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type: "integer",
							},
						},
					},
				},

				"_after": &openapi3.ParameterRef{
					Value: &openapi3.Parameter{
						Name:        "_after",
						In:          "query",
						Description: "Skip objects up to and including this ID",
						Schema: &openapi3.SchemaRef{
							Ref: "#/components/schemas/id",
						},
					},
				},
			},

			Responses: openapi3.Responses{
				// 204
				"no-content": &openapi3.ResponseRef{
					Value: &openapi3.Response{
						Description: P("No Content"),
					},
				},

				// 304
				"not-modified": &openapi3.ResponseRef{
					Value: &openapi3.Response{
						Description: P("Not Modified"),
					},
				},

				// 400
				"bad-request": &openapi3.ResponseRef{
					Value: &openapi3.Response{
						Description: P("Bad Request"),
						Content: openapi3.Content{
							"application/json": &openapi3.MediaType{
								Schema: &openapi3.SchemaRef{
									Ref: "#/components/schemas/error",
								},
							},
						},
					},
				},

				// 401
				"unauthorized": &openapi3.ResponseRef{
					Value: &openapi3.Response{
						Description: P("Unauthorized"),
						Content: openapi3.Content{
							"application/json": &openapi3.MediaType{
								Schema: &openapi3.SchemaRef{
									Ref: "#/components/schemas/error",
								},
							},
						},
					},
				},

				// 403
				"forbidden": &openapi3.ResponseRef{
					Value: &openapi3.Response{
						Description: P("Forbidden"),
						Content: openapi3.Content{
							"application/json": &openapi3.MediaType{
								Schema: &openapi3.SchemaRef{
									Ref: "#/components/schemas/error",
								},
							},
						},
					},
				},

				// 404
				"not-found": &openapi3.ResponseRef{
					Value: &openapi3.Response{
						Description: P("Not Found"),
						Content: openapi3.Content{
							"application/json": &openapi3.MediaType{
								Schema: &openapi3.SchemaRef{
									Ref: "#/components/schemas/error",
								},
							},
						},
					},
				},

				// 409
				"conflict": &openapi3.ResponseRef{
					Value: &openapi3.Response{
						Description: P("Conflict"),
						Content: openapi3.Content{
							"application/json": &openapi3.MediaType{
								Schema: &openapi3.SchemaRef{
									Ref: "#/components/schemas/error",
								},
							},
						},
					},
				},

				// 412
				"precondition-failed": &openapi3.ResponseRef{
					Value: &openapi3.Response{
						Description: P("Precondition Failed"),
						Content: openapi3.Content{
							"application/json": &openapi3.MediaType{
								Schema: &openapi3.SchemaRef{
									Ref: "#/components/schemas/error",
								},
							},
						},
					},
				},

				// 415
				"unsupported-media-type": &openapi3.ResponseRef{
					Value: &openapi3.Response{
						Description: P("Unsupported Media Type"),
						Content: openapi3.Content{
							"application/json": &openapi3.MediaType{
								Schema: &openapi3.SchemaRef{
									Ref: "#/components/schemas/error",
								},
							},
						},
					},
				},
			},

			Schemas: openapi3.Schemas{
				"id": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "string",
					},
				},

				"etag": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "string",
					},
				},

				"generation": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:   "integer",
						Format: "int64",
					},
				},

				"prefix": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: "string",
					},
				},

				"event-stream-object": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Title: "EventStream (Object)",
						Type:  "string",
						Extensions: map[string]any{
							"x-event-types": []string{
								"notModified",
								"initial",
								"update",
								"delete",
								"heartbeat",
								"error",
							},
						},
					},
				},

				"event-stream-list": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Title: "EventStream (List)",
						OneOf: openapi3.SchemaRefs{
							&openapi3.SchemaRef{
								Ref: "#/components/schemas/event-stream-list-full",
							},
							&openapi3.SchemaRef{
								Ref: "#/components/schemas/event-stream-list-diff",
							},
						},
					},
				},

				"event-stream-list-full": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Title: "EventStream (List; _stream=full)",
						Extensions: map[string]any{
							"x-event-types": []string{
								"notModified",
								"list",
								"heartbeat",
								"error",
							},
						},
					},
				},

				"event-stream-list-diff": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Title: "EventStream (List; _stream=diff)",
						Extensions: map[string]any{
							"x-event-types": []string{
								"notModified",
								"add",
								"remove",
								"update",
								"sync",
								"heartbeat",
								"error",
							},
						},
					},
				},

				"error": errorSchema,
			},
		},

		Servers: openapi3.Servers{
			&openapi3.Server{
				URL: baseURL,
			},
		},
	}

	if api.openAPI.info != nil {
		t.Info = api.openAPI.info
	}

	if api.authBasic {
		t.Components.SecuritySchemes["basicAuth"] = &openapi3.SecuritySchemeRef{
			Value: &openapi3.SecurityScheme{
				Type:   "http",
				Scheme: "basic",
			},
		}

		t.Security = append(t.Security, openapi3.SecurityRequirement{"basicAuth": []string{}})
	}

	if api.authBearer {
		t.Components.SecuritySchemes["bearerAuth"] = &openapi3.SecuritySchemeRef{
			Value: &openapi3.SecurityScheme{
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "secret-token:*",
			},
		}

		t.Security = append(t.Security, openapi3.SecurityRequirement{"bearerAuth": []string{}})
	}

	return t, nil
}

func (api *API) buildOpenAPIType(t *openapi3.T, cfg *config) error {
	t.Tags = append(t.Tags, &openapi3.Tag{
		Name: cfg.apiName,
	})

	{
		responseSchema, err := generateSchemaRef(cfg.typeOf)
		if err != nil {
			return jsrest.Errorf(jsrest.ErrInternalServerError, "generate schema ref failed (%w)", err)
		}

		responseSchema.Ref = ""
		responseSchema.Value.Title = fmt.Sprintf("%s Response", cfg.apiName)

		responseSchema.Value.Properties["id"] = &openapi3.SchemaRef{Ref: "#/components/schemas/id"}
		responseSchema.Value.Properties["etag"] = &openapi3.SchemaRef{Ref: "#/components/schemas/etag"}
		responseSchema.Value.Properties["generation"] = &openapi3.SchemaRef{Ref: "#/components/schemas/generation"}

		t.Components.Schemas[fmt.Sprintf("%s--response", cfg.apiName)] = responseSchema
	}

	{
		requestSchema, err := generateSchemaRef(cfg.typeOf)
		if err != nil {
			return jsrest.Errorf(jsrest.ErrInternalServerError, "generate schema ref failed (%w)", err)
		}

		requestSchema.Ref = ""
		delete(requestSchema.Value.Properties, "id")
		delete(requestSchema.Value.Properties, "etag")
		delete(requestSchema.Value.Properties, "generation")

		requestSchema.Value.Title = fmt.Sprintf("%s Request", cfg.apiName)

		t.Components.Schemas[fmt.Sprintf("%s--request", cfg.apiName)] = requestSchema
	}

	t.Components.RequestBodies[cfg.apiName] = &openapi3.RequestBodyRef{
		Value: &openapi3.RequestBody{
			Required: true,
			Content: openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{
						Ref: fmt.Sprintf("#/components/schemas/%s--request", cfg.apiName),
					},
				},
			},
		},
	}

	t.Components.Responses[cfg.apiName] = &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: P(fmt.Sprintf("OK: `%s`", cfg.apiName)),
			Headers: openapi3.Headers{
				"ETag": &openapi3.HeaderRef{
					Ref: "#/components/headers/etag",
				},
			},
			Content: openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{
						Ref: fmt.Sprintf("#/components/schemas/%s--response", cfg.apiName),
					},
				},
				"text/event-stream": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{
						Ref: "#/components/schemas/event-stream-object",
					},
				},
			},
		},
	}

	t.Components.Responses[fmt.Sprintf("%s--list", cfg.apiName)] = &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: P(fmt.Sprintf("OK: List of `%s`", cfg.apiName)),
			Headers: openapi3.Headers{
				"ETag": &openapi3.HeaderRef{
					Ref: "#/components/headers/etag",
				},
			},
			Content: openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: "array",
							Items: &openapi3.SchemaRef{
								Ref: fmt.Sprintf("#/components/schemas/%s--response", cfg.apiName),
							},
						},
					},
				},
				"text/event-stream": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{
						Ref: "#/components/schemas/event-stream-list",
					},
				},
			},
		},
	}

	paths := path.ListType(cfg.typeOf)
	sorts := []any{}
	filters := openapi3.Parameters{}

	for _, pth := range paths {
		sorts = append(sorts, fmt.Sprintf("+%s", pth), fmt.Sprintf("-%s", pth))

		pthSchema, err := generateSchemaRef(path.GetFieldType(cfg.typeOf, pth))
		if err != nil {
			return jsrest.Errorf(jsrest.ErrInternalServerError, "generate schema ref failed (%w)", err)
		}

		filters = append(filters, openapi3.Parameters{
			&openapi3.ParameterRef{
				Value: &openapi3.Parameter{
					Name:        pth,
					In:          "query",
					Description: fmt.Sprintf("Filter list by `%s` equal to", pth),
					Schema:      pthSchema,
				},
			},

			&openapi3.ParameterRef{
				Value: &openapi3.Parameter{
					Name:        fmt.Sprintf("%s[gt]", pth),
					In:          "query",
					Description: fmt.Sprintf("Filter list by `%s` greater than", pth),
					Schema:      pthSchema,
				},
			},

			&openapi3.ParameterRef{
				Value: &openapi3.Parameter{
					Name:        fmt.Sprintf("%s[gte]", pth),
					In:          "query",
					Description: fmt.Sprintf("Filter list by `%s` greater than or equal to", pth),
					Schema:      pthSchema,
				},
			},

			&openapi3.ParameterRef{
				Value: &openapi3.Parameter{
					Name:        fmt.Sprintf("%s[hp]", pth),
					In:          "query",
					Description: fmt.Sprintf("Filter list by `%s` has prefix", pth),
					Schema: &openapi3.SchemaRef{
						Ref: "#/components/schemas/prefix",
					},
				},
			},

			&openapi3.ParameterRef{
				Value: &openapi3.Parameter{
					Name:        fmt.Sprintf("%s[in]", pth),
					In:          "query",
					Description: fmt.Sprintf("Filter list by `%s` one of", pth),
					Explode:     P(false),
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type:  "array",
							Items: pthSchema,
						},
					},
				},
			},

			&openapi3.ParameterRef{
				Value: &openapi3.Parameter{
					Name:        fmt.Sprintf("%s[lt]", pth),
					In:          "query",
					Description: fmt.Sprintf("Filter list by `%s` less than", pth),
					Schema:      pthSchema,
				},
			},

			&openapi3.ParameterRef{
				Value: &openapi3.Parameter{
					Name:        fmt.Sprintf("%s[lte]", pth),
					In:          "query",
					Description: fmt.Sprintf("Filter list by `%s` less than or equal to", pth),
					Schema:      pthSchema,
				},
			},
		}...)
	}

	t.Paths[fmt.Sprintf("/%s", cfg.apiName)] = &openapi3.PathItem{
		Get: &openapi3.Operation{
			Tags:    []string{cfg.apiName},
			Summary: fmt.Sprintf("List %s objects", cfg.apiName),
			Parameters: append(filters, openapi3.Parameters{
				&openapi3.ParameterRef{
					Ref: "#/components/headers/if-none-match",
				},
				&openapi3.ParameterRef{
					Ref: "#/components/parameters/_stream",
				},
				&openapi3.ParameterRef{
					Ref: "#/components/parameters/_limit",
				},
				&openapi3.ParameterRef{
					Ref: "#/components/parameters/_offset",
				},
				&openapi3.ParameterRef{
					Ref: "#/components/parameters/_after",
				},
				&openapi3.ParameterRef{
					Value: &openapi3.Parameter{
						Name:        "_sort",
						In:          "query",
						Description: "Direction (`+` ascending or `-` descending) and field path to sort by",
						Explode:     P(true),
						Schema: &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type: "array",
								Items: &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Type: "enum",
										Enum: sorts,
									},
								},
							},
						},
					},
				},
			}...),
			Responses: openapi3.Responses{
				"200": &openapi3.ResponseRef{
					Ref: fmt.Sprintf("#/components/responses/%s--list", cfg.apiName),
				},
				"304": &openapi3.ResponseRef{
					Ref: "#/components/responses/not-modified",
				},
				"400": &openapi3.ResponseRef{
					Ref: "#/components/responses/bad-request",
				},
				"401": &openapi3.ResponseRef{
					Ref: "#/components/responses/unauthorized",
				},
				"403": &openapi3.ResponseRef{
					Ref: "#/components/responses/forbidden",
				},
			},
		},

		Post: &openapi3.Operation{
			Tags:    []string{cfg.apiName},
			Summary: fmt.Sprintf("Create new %s object", cfg.apiName),
			Parameters: openapi3.Parameters{
				&openapi3.ParameterRef{
					Ref: "#/components/headers/idempotency-key",
				},
			},
			RequestBody: &openapi3.RequestBodyRef{
				Ref: fmt.Sprintf("#/components/requestBodies/%s", cfg.apiName),
			},
			Responses: openapi3.Responses{
				"200": &openapi3.ResponseRef{
					Ref: fmt.Sprintf("#/components/responses/%s", cfg.apiName),
				},
				"400": &openapi3.ResponseRef{
					Ref: "#/components/responses/bad-request",
				},
				"401": &openapi3.ResponseRef{
					Ref: "#/components/responses/unauthorized",
				},
				"403": &openapi3.ResponseRef{
					Ref: "#/components/responses/forbidden",
				},
				"409": &openapi3.ResponseRef{
					Ref: "#/components/responses/conflict",
				},
				"415": &openapi3.ResponseRef{
					Ref: "#/components/responses/unsupported-media-type",
				},
			},
		},
	}

	t.Paths[fmt.Sprintf("/%s/{id}", cfg.apiName)] = &openapi3.PathItem{
		Parameters: openapi3.Parameters{
			&openapi3.ParameterRef{
				Ref: "#/components/parameters/id",
			},
		},

		Get: &openapi3.Operation{
			Tags:    []string{cfg.apiName},
			Summary: fmt.Sprintf("Get %s object", cfg.apiName),
			Parameters: openapi3.Parameters{
				&openapi3.ParameterRef{
					Ref: "#/components/headers/if-none-match",
				},
			},
			Responses: openapi3.Responses{
				"200": &openapi3.ResponseRef{
					Ref: fmt.Sprintf("#/components/responses/%s", cfg.apiName),
				},
				"304": &openapi3.ResponseRef{
					Ref: "#/components/responses/not-modified",
				},
				"400": &openapi3.ResponseRef{
					Ref: "#/components/responses/bad-request",
				},
				"401": &openapi3.ResponseRef{
					Ref: "#/components/responses/unauthorized",
				},
				"403": &openapi3.ResponseRef{
					Ref: "#/components/responses/forbidden",
				},
				"404": &openapi3.ResponseRef{
					Ref: "#/components/responses/not-found",
				},
			},
		},

		Put: &openapi3.Operation{
			Tags:    []string{cfg.apiName},
			Summary: fmt.Sprintf("Replace %s object", cfg.apiName),
			Parameters: openapi3.Parameters{
				&openapi3.ParameterRef{
					Ref: "#/components/headers/if-match",
				},
				&openapi3.ParameterRef{
					Ref: "#/components/headers/idempotency-key",
				},
			},
			RequestBody: &openapi3.RequestBodyRef{
				Ref: fmt.Sprintf("#/components/requestBodies/%s", cfg.apiName),
			},
			Responses: openapi3.Responses{
				"200": &openapi3.ResponseRef{
					Ref: fmt.Sprintf("#/components/responses/%s", cfg.apiName),
				},
				"400": &openapi3.ResponseRef{
					Ref: "#/components/responses/bad-request",
				},
				"401": &openapi3.ResponseRef{
					Ref: "#/components/responses/unauthorized",
				},
				"403": &openapi3.ResponseRef{
					Ref: "#/components/responses/forbidden",
				},
				"404": &openapi3.ResponseRef{
					Ref: "#/components/responses/not-found",
				},
				"409": &openapi3.ResponseRef{
					Ref: "#/components/responses/conflict",
				},
				"412": &openapi3.ResponseRef{
					Ref: "#/components/responses/precondition-failed",
				},
				"415": &openapi3.ResponseRef{
					Ref: "#/components/responses/unsupported-media-type",
				},
			},
		},

		Patch: &openapi3.Operation{
			Tags:    []string{cfg.apiName},
			Summary: fmt.Sprintf("Update %s object", cfg.apiName),
			Parameters: openapi3.Parameters{
				&openapi3.ParameterRef{
					Ref: "#/components/headers/if-match",
				},
				&openapi3.ParameterRef{
					Ref: "#/components/headers/idempotency-key",
				},
			},
			RequestBody: &openapi3.RequestBodyRef{
				Ref: fmt.Sprintf("#/components/requestBodies/%s", cfg.apiName),
			},
			Responses: openapi3.Responses{
				"200": &openapi3.ResponseRef{
					Ref: fmt.Sprintf("#/components/responses/%s", cfg.apiName),
				},
				"400": &openapi3.ResponseRef{
					Ref: "#/components/responses/bad-request",
				},
				"401": &openapi3.ResponseRef{
					Ref: "#/components/responses/unauthorized",
				},
				"403": &openapi3.ResponseRef{
					Ref: "#/components/responses/forbidden",
				},
				"404": &openapi3.ResponseRef{
					Ref: "#/components/responses/not-found",
				},
				"409": &openapi3.ResponseRef{
					Ref: "#/components/responses/conflict",
				},
				"412": &openapi3.ResponseRef{
					Ref: "#/components/responses/precondition-failed",
				},
				"415": &openapi3.ResponseRef{
					Ref: "#/components/responses/unsupported-media-type",
				},
			},
		},

		Delete: &openapi3.Operation{
			Tags:    []string{cfg.apiName},
			Summary: fmt.Sprintf("Delete %s object", cfg.apiName),
			Parameters: openapi3.Parameters{
				&openapi3.ParameterRef{
					Ref: "#/components/headers/if-match",
				},
				&openapi3.ParameterRef{
					Ref: "#/components/headers/idempotency-key",
				},
			},
			Responses: openapi3.Responses{
				"204": &openapi3.ResponseRef{
					Ref: "#/components/responses/no-content",
				},
				"400": &openapi3.ResponseRef{
					Ref: "#/components/responses/bad-request",
				},
				"401": &openapi3.ResponseRef{
					Ref: "#/components/responses/unauthorized",
				},
				"403": &openapi3.ResponseRef{
					Ref: "#/components/responses/forbidden",
				},
				"404": &openapi3.ResponseRef{
					Ref: "#/components/responses/not-found",
				},
				"409": &openapi3.ResponseRef{
					Ref: "#/components/responses/conflict",
				},
				"412": &openapi3.ResponseRef{
					Ref: "#/components/responses/precondition-failed",
				},
			},
		},
	}

	return nil
}

func (api *API) requestBaseURL(r *http.Request) (string, error) {
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}

	host, err := idna.ToUnicode(r.Host)
	if err != nil {
		return "", jsrest.Errorf(jsrest.ErrInternalServerError, "unicode hostname conversion failed (%w)", err)
	}

	i := strings.Index(r.RequestURI, "/_openapi")
	if i == -1 {
		return "", jsrest.Errorf(jsrest.ErrInternalServerError, "missing /_openapi in URL")
	}

	path := r.RequestURI[:i]

	return fmt.Sprintf("%s://%s%s", scheme, host, path), nil
}

func generateSchemaRef(t reflect.Type) (*openapi3.SchemaRef, error) {
	gen := openapi3gen.NewGenerator()

	schemaRef, err := gen.GenerateSchemaRef(t)
	if err != nil {
		return nil, err
	}

	for ref := range gen.SchemaRefs {
		ref.Ref = ""
	}

	return schemaRef, nil
}
