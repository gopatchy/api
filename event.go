package patchy

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"runtime/metrics"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

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
	events             []*event
}

type event struct {
	Time       string         `json:"time"`
	SampleRate int64          `json:"samplerate"`
	Data       map[string]any `json:"data"`
}

func (api *API) AddEventTarget(url string, headers map[string]string, eventsPerSecond, writePeriodSeconds float64) {
	target := &eventTarget{
		client:             resty.New().SetBaseURL(url).SetHeaders(headers),
		eventsPerSecond:    eventsPerSecond,
		writePeriodSeconds: writePeriodSeconds,
		done:               make(chan bool),
	}

	go target.writeLoop(&api.eventState)

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
			target.events = append(target.events, &ev2)
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
			"httpProto":     r.Proto,
			"requestHost":   r.Host,
			"requestMethod": r.Method,
			"requestPath":   r.URL.Path,
			"remoteAddr":    r.RemoteAddr,
			"responseCode":  200,
			"durationMS":    time.Since(start).Milliseconds(),
		},
	}

	if err != nil {
		hErr := jsrest.GetHTTPError(err)
		if hErr != nil {
			ev.Data["responseCode"] = hErr.Code
		}

		ev.Data["responseError"] = err.Error()
	}

	spanID := ctx.Value(ContextSpanID)

	if spanID != nil {
		ev.Data["spanID"] = spanID.(string)
	}

	ev.addMetrics()
	ev.addRUsage()

	data := ctx.Value(ContextEventData)

	if data != nil {
		for k, v := range data.(map[string]any) {
			ev.Data[k] = v
		}
	}

	return ev
}

func (target *eventTarget) writeLoop(es *eventState) {
	t := time.NewTicker(time.Duration(target.writePeriodSeconds * float64(time.Second)))
	defer t.Stop()

	for {
		select {
		case <-target.done:
			target.write(es)
			return

		case <-t.C:
			target.write(es)
		}
	}
}

func (target *eventTarget) write(es *eventState) {
	es.mu.Lock()
	events := target.events
	target.events = nil
	es.mu.Unlock()

	if len(events) == 0 {
		return
	}

	resp, err := target.client.R().
		SetBody(events).
		Post("")
	if err != nil {
		log.Printf("HTTP %s", err)
		return
	}

	if resp.IsError() {
		log.Printf("HTTP %d %s: %s", resp.StatusCode(), resp.Status(), resp.String())
		return
	}
}

func (ev *event) addMetrics() {
	descs := metrics.All()

	samples := make([]metrics.Sample, len(descs))
	for i := range samples {
		samples[i].Name = descs[i].Name
	}

	metrics.Read(samples)

	for _, sample := range samples {
		name := convertMetricName(sample.Name)

		switch sample.Value.Kind() { //nolint:exhaustive
		case metrics.KindUint64:
			ev.Data[name] = sample.Value.Uint64()
		case metrics.KindFloat64:
			ev.Data[name] = sample.Value.Float64()
		}
	}
}

func (ev *event) addRUsage() {
	rusage := &syscall.Rusage{}

	err := syscall.Getrusage(syscall.RUSAGE_SELF, rusage)
	if err != nil {
		panic(err)
	}

	ev.Data["rUsageUTime"] = time.Duration(rusage.Utime.Nano()).Seconds()
	ev.Data["rUsageSTime"] = time.Duration(rusage.Stime.Nano()).Seconds()
}

func convertMetricName(in string) string {
	upperNext := false

	in = strings.TrimLeft(in, "/")

	ret := strings.Builder{}

	for _, r := range in {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			upperNext = true
			continue
		}

		if upperNext {
			r = unicode.ToUpper(r)
			upperNext = false
		}

		ret.WriteRune(r)
	}

	return ret.String()
}
