package patchy

import (
	"bytes"
	"embed"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"text/template"
	"time"

	"cloud.google.com/go/civil"
	"github.com/gopatchy/jsrest"
	"github.com/gopatchy/path"
	"github.com/julienschmidt/httprouter"
)

//go:embed templates/*
var templateFS embed.FS

var templates = template.Must(
	template.New("templates").
		Funcs(template.FuncMap{
			"add":      add,
			"padRight": padRight,
		}).
		ParseFS(templateFS, "templates/*"))

type templateInput struct {
	Info       *OpenAPIInfo
	Form       url.Values
	Types      []*templateType
	APIs       []*apiType
	UsesTime   bool
	UsesCivil  bool
	URLPrefix  string
	AuthBasic  bool
	AuthBearer bool
}

type apiType struct {
	NameLower      string // "homeaddress"
	NameUpperCamel string // "HomeAddress"
	TypeUpperCamel string // "AddressType"

	typeOf reflect.Type
}

type templateType struct {
	TopLevel bool

	TypeUpperCamel string // "AddressType"

	Fields            []*templateField
	FieldNameMaxLen   int
	FieldGoTypeMaxLen int

	typeOf reflect.Type
}

type templateField struct {
	NameLower      string // "streetname"
	NameUpperCamel string // "StreetName"
	NameLowerCamel string // "streetName"
	GoType         string // "bool"
	TSType         string // "boolean"
	Optional       bool
}

func (api *API) registerTemplates() {
	api.router.GET("/_client.go", api.writeTemplate("client.go"))
	api.router.GET("/_client.ts", api.writeTemplate("client.ts"))
}

func (api *API) writeTemplate(name string) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()

		api.SetEventData(ctx,
			"operation", "template",
			"template", name,
		)

		input := &templateInput{
			Info:       api.openAPI.info,
			Form:       r.Form,
			URLPrefix:  api.prefix,
			AuthBasic:  api.authBasic,
			AuthBearer: api.authBearer,
		}

		typeQueue := []*templateType{}
		typesDone := map[reflect.Type]bool{}

		for _, apiName := range api.names() {
			cfg := api.registry[apiName]

			typeQueue = append(typeQueue, &templateType{
				TopLevel: true,
				typeOf:   cfg.typeOf,
			})

			input.APIs = append(input.APIs, &apiType{
				NameLower:      cfg.apiName,
				NameUpperCamel: cfg.camelName,
				TypeUpperCamel: upperFirst(cfg.typeOf.Name()),
				typeOf:         cfg.typeOf,
			})
		}

		for len(typeQueue) > 0 {
			tt := typeQueue[0]
			typeQueue = typeQueue[1:]

			if typesDone[tt.typeOf] {
				continue
			}

			typesDone[tt.typeOf] = true

			tt.TypeUpperCamel = upperFirst(tt.typeOf.Name())

			path.WalkType(tt.typeOf, func(_ string, parts []string, field reflect.StructField) {
				typeOf := path.MaybeIndirectType(field.Type)

				elemType := typeOf
				if elemType.Kind() == reflect.Slice {
					elemType = path.MaybeIndirectType(elemType.Elem())
				}

				if len(parts) > 1 || parts[0] == "" {
					return
				}

				if tt.TopLevel && (parts[0] == "id" ||
					parts[0] == "etag" ||
					parts[0] == "generation") {
					return
				}

				tf := &templateField{
					NameLower:      parts[0],
					NameUpperCamel: upperFirst(parts[0]),
					NameLowerCamel: lowerFirst(parts[0]),
					GoType:         goType(field.Type),
					TSType:         tsType(field.Type),
					Optional:       field.Type.Kind() == reflect.Pointer,
				}

				if strings.EqualFold(tf.NameUpperCamel, field.Name) {
					tf.NameUpperCamel = field.Name
					tf.NameLowerCamel = lowerFirst(field.Name)
				}

				if elemType.Kind() == reflect.Struct && elemType != reflect.TypeOf(time.Time{}) && elemType != reflect.TypeOf(civil.Date{}) {
					typeQueue = append(typeQueue, &templateType{
						typeOf: elemType,
					})
				}

				if len(tf.NameLower) > tt.FieldNameMaxLen {
					tt.FieldNameMaxLen = len(tf.NameLower)
				}

				if len(tf.GoType) > tt.FieldGoTypeMaxLen {
					tt.FieldGoTypeMaxLen = len(tf.GoType)
				}

				switch typeOf {
				case path.TimeTimeType:
					input.UsesTime = true

				case path.CivilDateType:
					input.UsesCivil = true
				}

				tt.Fields = append(tt.Fields, tf)
			})

			input.Types = append(input.Types, tt)
		}

		// Buffer this so we can handle the error before sending any template output
		buf := &bytes.Buffer{}

		err := templates.ExecuteTemplate(buf, name, input)
		if err != nil {
			err = jsrest.Errorf(jsrest.ErrInternalServerError, "execute template failed (%w)", err)
			jsrest.WriteError(w, err)

			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		_, _ = buf.WriteTo(w)
	}
}

func add(a, b int) int {
	return a + b
}

func padRight(in string, l int) string {
	return fmt.Sprintf(fmt.Sprintf("%%-%ds", l), in)
}

func lowerFirst(in string) string {
	return strings.ToLower(in[:1]) + in[1:]
}

func upperFirst(in string) string {
	return strings.ToUpper(in[:1]) + in[1:]
}

func goType(t reflect.Type) string {
	elemType := path.MaybeIndirectType(t)

	if elemType.Kind() != reflect.Struct || elemType == path.TimeTimeType || elemType == path.CivilDateType {
		return elemType.String()
	}

	return upperFirst(elemType.Name())
}

func tsType(t reflect.Type) string {
	elemType := path.MaybeIndirectType(t)

	if elemType == path.TimeTimeType || elemType == path.CivilDateType {
		return "string"
	}

	switch elemType.Kind() { //nolint:exhaustive
	case reflect.Slice:
		return fmt.Sprintf("%s[]", tsType(elemType.Elem()))

	case reflect.Int:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int64:
		fallthrough
	case reflect.Uint:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint64:
		fallthrough
	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		return "number"

	case reflect.Bool:
		return "boolean"

	case reflect.Struct:
		return goType(elemType)

	case reflect.Interface:
		return "any"

	default:
		return "string"
	}
}
