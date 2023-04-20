package patchy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gopatchy/jsrest"
	"github.com/gopatchy/metadata"
	"github.com/vfaronov/httpheader"
)

var ErrStreamingNotSupported = errors.New("streaming not supported")

func (api *API) streamGet(cfg *config, id string, w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	opts := parseGetOpts(r)

	if _, ok := w.(http.Flusher); !ok {
		return jsrest.Errorf(jsrest.ErrBadRequest, "stream failed (%w)", ErrStreamingNotSupported)
	}

	gsi, err := api.streamGetInt(ctx, cfg, id)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "read failed: %s (%w)", id, err)
	}

	defer gsi.Close()

	w.Header().Set("Content-Type", "text/event-stream")

	err = api.streamGetWrite(ctx, w, gsi.ch, opts)
	if err != nil {
		_ = writeEvent(w, "error", nil, jsrest.ToJSONError(err), true)
		return nil
	}

	return nil
}

func (api *API) streamGetWrite(ctx context.Context, w http.ResponseWriter, ch <-chan any, opts *GetOpts) error {
	first := true
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return nil

		case obj, ok := <-ch:
			if !ok {
				err := writeEvent(w, "delete", nil, nil, true)
				if err != nil {
					return jsrest.Errorf(jsrest.ErrInternalServerError, "write delete failed (%w)", err)
				}

				return nil
			}

			eventType := "update"

			md := metadata.GetMetadata(obj)
			gen := fmt.Sprintf("generation:%d", md.Generation)

			if first {
				first = false
				eventType = "initial"

				if httpheader.MatchWeak(opts.IfNoneMatch, httpheader.EntityTag{Opaque: md.ETag}) ||
					httpheader.MatchWeak(opts.IfNoneMatch, httpheader.EntityTag{Opaque: gen}) {
					err := writeEvent(w, "notModified", map[string]string{"id": md.ETag}, nil, true)
					if err != nil {
						return jsrest.Errorf(jsrest.ErrInternalServerError, "write update failed (%w)", err)
					}

					first = false

					continue
				}
			}

			err := writeEvent(w, eventType, map[string]string{"id": md.ETag}, obj, true)
			if err != nil {
				return jsrest.Errorf(jsrest.ErrInternalServerError, "write update failed (%w)", err)
			}

		case <-ticker.C:
			err := writeEvent(w, "heartbeat", nil, nil, true)
			if err != nil {
				return jsrest.Errorf(jsrest.ErrInternalServerError, "write heartbeat failed (%w)", err)
			}
		}
	}
}
