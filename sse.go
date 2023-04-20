package patchy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gopatchy/jsrest"
)

func writeEvent(w http.ResponseWriter, event string, params map[string]string, obj any, flush bool) error {
	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "event: %s\n", event)

	for k, v := range params {
		fmt.Fprintf(buf, "%s: %s\n", k, v)
	}

	if obj != nil {
		buf.WriteString("data: ")

		enc := json.NewEncoder(buf)

		err := enc.Encode(obj)
		if err != nil {
			return jsrest.Errorf(jsrest.ErrInternalServerError, "encode JSON failed (%w)", err)
		}
	}

	buf.WriteString("\n")

	_, err := buf.WriteTo(w)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "write event failed (%w)", err)
	}

	if flush {
		w.(http.Flusher).Flush()
	}

	return nil
}
