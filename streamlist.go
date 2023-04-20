package patchy

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gopatchy/jsrest"
	"github.com/gopatchy/metadata"
	"github.com/vfaronov/httpheader"
)

func (api *API) streamList(cfg *config, w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	if _, ok := w.(http.Flusher); !ok {
		return jsrest.Errorf(jsrest.ErrBadRequest, "stream failed (%w)", ErrStreamingNotSupported)
	}

	opts, err := api.parseListOpts(r)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrBadRequest, "parse list parameters failed (%w)", err)
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Stream-Format", opts.Stream)

	switch opts.Stream {
	case "full":
		err = api.streamListFull(ctx, cfg, w, opts)
		if err != nil {
			_ = writeEvent(w, "error", nil, jsrest.ToJSONError(err), true)
		}

		return nil

	case "diff":
		err = api.streamListDiff(ctx, cfg, w, opts)
		if err != nil {
			_ = writeEvent(w, "error", nil, jsrest.ToJSONError(err), true)
		}

		return nil

	default:
		return jsrest.Errorf(jsrest.ErrBadRequest, "_stream=%s (%w)", opts.Stream, ErrInvalidStreamFormat)
	}
}

func (api *API) streamListFull(ctx context.Context, cfg *config, w http.ResponseWriter, opts *ListOpts) error {
	// TODO: Add query condition pushdown
	lsi, err := api.streamListInt(ctx, cfg, opts)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "read list failed (%w)", err)
	}
	defer lsi.Close()

	ticker := time.NewTicker(5 * time.Second)
	ifNoneMatch := opts.IfNoneMatch
	previousETag := ""

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			err = writeEvent(w, "heartbeat", nil, nil, true)
			if err != nil {
				return jsrest.Errorf(jsrest.ErrInternalServerError, "write heartbeat failed (%w)", err)
			}

		case list := <-lsi.Chan():
			etag, err := hashList(list)
			if err != nil {
				return jsrest.Errorf(jsrest.ErrInternalServerError, "hash list failed (%w)", err)
			}

			if ifNoneMatch != nil && httpheader.MatchWeak(opts.IfNoneMatch, httpheader.EntityTag{Opaque: etag}) {
				ifNoneMatch = nil

				err = writeEvent(w, "notModified", map[string]string{"id": etag}, nil, true)
				if err != nil {
					return jsrest.Errorf(jsrest.ErrInternalServerError, "write list failed (%w)", err)
				}

				continue
			}

			ifNoneMatch = nil

			if previousETag == etag {
				continue
			}

			previousETag = etag

			err = writeEvent(w, "list", map[string]string{"id": etag}, list, true)
			if err != nil {
				return jsrest.Errorf(jsrest.ErrInternalServerError, "write list failed (%w)", err)
			}
		}
	}
}

type listEntry struct {
	pos int
	obj any
}

func (api *API) streamListDiff(ctx context.Context, cfg *config, w http.ResponseWriter, opts *ListOpts) error {
	lsi, err := api.streamListInt(ctx, cfg, opts)
	if err != nil {
		return jsrest.Errorf(jsrest.ErrInternalServerError, "read list failed (%w)", err)
	}
	defer lsi.Close()

	last := map[string]*listEntry{}

	ticker := time.NewTicker(5 * time.Second)
	ifNoneMatch := opts.IfNoneMatch
	previousETag := ""

	for {
		select {
		case <-ticker.C:
			err = writeEvent(w, "heartbeat", nil, nil, true)
			if err != nil {
				return jsrest.Errorf(jsrest.ErrInternalServerError, "write heartbeat failed (%w)", err)
			}

			continue

		case <-ctx.Done():
			return nil

		case list := <-lsi.Chan():
			// Don't do anything if the list hasn't changed (can't trigger first time)
			etag, err := hashList(list)
			if err != nil {
				return jsrest.Errorf(jsrest.ErrInternalServerError, "hash list failed (%w)", err)
			}

			if previousETag == etag {
				continue
			}

			previousETag = etag

			// Build a map of the current list
			cur := map[string]*listEntry{}

			for pos, obj := range list {
				objMD := metadata.GetMetadata(obj)

				cur[objMD.ID] = &listEntry{
					pos: pos,
					obj: obj,
				}
			}

			// Short-circuit with notModified, if appropriate
			tmpIfNoneMatch := ifNoneMatch
			ifNoneMatch = nil

			if tmpIfNoneMatch != nil && httpheader.MatchWeak(tmpIfNoneMatch, httpheader.EntityTag{Opaque: etag}) {
				last = cur

				err = writeEvent(w, "notModified", map[string]string{"id": etag}, nil, true)
				if err != nil {
					return jsrest.Errorf(jsrest.ErrInternalServerError, "write list failed (%w)", err)
				}

				continue
			}

			// If we reach here, the list has actually changed from the client's view

			// remove events have to go out before add/update events, for ordering
			for id, lastEntry := range last {
				if cur[id] != nil {
					continue
				}

				err = writeEvent(w, "remove", map[string]string{"old-position": strconv.Itoa(lastEntry.pos)}, nil, false)
				if err != nil {
					return jsrest.Errorf(jsrest.ErrInternalServerError, "write remove failed (%w)", err)
				}
			}

			// Use the list instead of the map because order matters
			for pos, obj := range list {
				objMD := metadata.GetMetadata(obj)

				lastEntry := last[objMD.ID]
				if lastEntry == nil {
					err = writeEvent(w, "add", map[string]string{"new-position": strconv.Itoa(pos)}, obj, false)
					if err != nil {
						return jsrest.Errorf(jsrest.ErrInternalServerError, "write add failed (%w)", err)
					}
				} else {
					lastMD := metadata.GetMetadata(lastEntry.obj)
					if objMD.ETag != lastMD.ETag {
						params := map[string]string{
							"old-position": strconv.Itoa(lastEntry.pos),
							"new-position": strconv.Itoa(pos),
						}

						err = writeEvent(w, "update", params, obj, false)
						if err != nil {
							return jsrest.Errorf(jsrest.ErrInternalServerError, "write update failed (%w)", err)
						}
					}
				}
			}

			last = cur

			err = writeEvent(w, "sync", map[string]string{"id": etag}, nil, true)
			if err != nil {
				return jsrest.Errorf(jsrest.ErrInternalServerError, "write sync failed (%w)", err)
			}
		}
	}
}
