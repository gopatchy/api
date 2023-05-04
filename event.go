package patchy

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gopatchy/jsrest"
)

type eventState struct {
	baseEventData map[string]any

	targets []*eventTarget

	lastEvent              time.Time
	successEventsPerSecond float64
	failureEventsPerSecond float64

	mu sync.Mutex
}

type eventTarget struct {
	client             *resty.Client
	eventsPerSecond    float64
	writePeriodSeconds float64
	done               chan bool

	// TODO: Remove double locking
	events   []*event
	eventsMu sync.Mutex
}

type event struct {
	Time       string         `json:"time"`
	SampleRate int64          `json:"samplerate"`
	Data       map[string]any `json:"data"`
}

func (api *API) AddEventTarget(url string, headers map[string]string, eventsPerSecond, writePeriodSeconds float64) {
	target := &eventTarget{
		// TODO: Enable compression
		client:             resty.New().SetBaseURL(url).SetHeaders(headers),
		eventsPerSecond:    eventsPerSecond,
		writePeriodSeconds: writePeriodSeconds,
		done:               make(chan bool),
	}

	go target.writeLoop()

	api.eventState.mu.Lock()
	defer api.eventState.mu.Unlock()

	api.eventState.targets = append(api.eventState.targets, target)
}

func (api *API) AddBaseEventData(k string, v any) {
	api.eventState.mu.Lock()
	defer api.eventState.mu.Unlock()

	api.eventState.baseEventData[k] = v
}

func (api *API) AddEventData(ctx context.Context, k string, v any) {
	data := ctx.Value(ContextEventData)
	if data == nil {
		return
	}

	data.(map[string]any)[k] = v
}

func (api *API) closeEventTargets() {
	api.eventState.mu.Lock()
	defer api.eventState.mu.Unlock()

	for _, target := range api.eventState.targets {
		close(target.done)
	}
}

func (api *API) writeEvent(ctx context.Context, r *http.Request, err error, start time.Time) {
	ev := api.buildEvent(ctx, r, err, start)
	rnd := rand.Float64() //nolint:gosec

	api.eventState.mu.Lock()
	defer api.eventState.mu.Unlock()

	api.updateEventRates(err)
	eventsPerSecond := api.eventState.successEventsPerSecond + api.eventState.failureEventsPerSecond

	for _, target := range api.eventState.targets {
		// TODO: Reserve some portion for errors if we're over
		prob := target.eventsPerSecond / eventsPerSecond

		if rnd < prob {
			ev2 := *ev
			ev2.SampleRate = int64(1 / prob)

			target.eventsMu.Lock()
			target.events = append(target.events, &ev2)
			target.eventsMu.Unlock()
		}
	}
}

func (api *API) updateEventRates(err error) {
	bucket := &api.eventState.successEventsPerSecond
	if err != nil {
		bucket = &api.eventState.failureEventsPerSecond
	}

	*bucket++

	now := time.Now()

	if !api.eventState.lastEvent.IsZero() {
		*bucket /= (now.Sub(api.eventState.lastEvent).Seconds() + 1)
	}

	api.eventState.lastEvent = now
}

func (api *API) buildEvent(ctx context.Context, r *http.Request, err error, start time.Time) *event {
	ev := &event{
		Time: time.Now().Format(time.RFC3339Nano),
		Data: map[string]any{
			"proto":                r.Proto,
			"host":                 r.Host,
			"method":               r.Method,
			"route":                r.URL.Path,
			"remote_addr":          r.RemoteAddr,
			"response.status_code": 200,
			"error":                nil,
			"duration_ms":          time.Since(start).Milliseconds(),
		},
	}

	// TODO: Add process metrics (CPU, memory, IO?)

	if err != nil {
		hErr := jsrest.GetHTTPError(err)
		if hErr != nil {
			ev.Data["response.status_code"] = hErr.Code
		}

		ev.Data["error"] = err.Error()
	}

	spanID := ctx.Value(ContextSpanID)

	if spanID != nil {
		ev.Data["trace.span_id"] = spanID.(string)
	}

	data := ctx.Value(ContextEventData)

	if data != nil {
		for k, v := range data.(map[string]any) {
			ev.Data[k] = v
		}
	}

	return ev
}

func (target *eventTarget) writeLoop() {
	t := time.NewTicker(time.Duration(target.writePeriodSeconds * float64(time.Second)))
	defer t.Stop()

	for {
		select {
		case <-target.done:
			target.write()
			return

		case <-t.C:
			target.write()
		}
	}
}

func (target *eventTarget) write() {
	target.eventsMu.Lock()
	events := target.events
	target.events = nil
	target.eventsMu.Unlock()

	if len(events) == 0 {
		return
	}

	resp, err := target.client.R().
		SetBody(events).
		Post("")
	if err != nil {
		panic(err)
	}

	if resp.IsError() {
		log.Printf("HTTP %d %s: %s", resp.StatusCode(), resp.Status(), resp.String())
	}
}
