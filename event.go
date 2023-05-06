package patchy

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"runtime/debug"
	"runtime/metrics"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/go-resty/resty/v2"
)

type eventState struct {
	targets []*eventTarget
	hooks   []EventHook

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
	events             []*Event
}

type Event struct {
	start time.Time

	Time       string         `json:"time"`
	SampleRate int64          `json:"samplerate"`
	Data       map[string]any `json:"data"`
}

type EventHook func(context.Context, *Event)

func (api *API) AddEventTarget(url string, headers map[string]string, eventsPerSecond, writePeriodSeconds float64) {
	target := &eventTarget{
		client:             resty.New().SetBaseURL(url).SetHeaders(headers),
		eventsPerSecond:    eventsPerSecond,
		writePeriodSeconds: writePeriodSeconds,
		done:               make(chan bool),
	}

	go api.eventState.flushLoop(target)

	api.eventState.mu.Lock()
	defer api.eventState.mu.Unlock()

	api.eventState.targets = append(api.eventState.targets, target)
}

func (api *API) AddEventHook(hook EventHook) {
	api.eventState.mu.Lock()
	defer api.eventState.mu.Unlock()

	api.eventState.hooks = append(api.eventState.hooks, hook)
}

func (api *API) SetEventData(ctx context.Context, vals ...any) {
	ev := ctx.Value(ContextEvent)

	if ev == nil {
		return
	}

	ev.(*Event).Set(vals...)
}

func (api *API) Log(ctx context.Context, eventType string, vals ...any) {
	if len(vals)%2 != 0 {
		panic(vals)
	}

	ev := api.newEvent(eventType, vals...)
	api.eventState.WriteEvent(ctx, ev)

	parts := []string{
		fmt.Sprintf("type=%s", eventType),
	}

	for i := 0; i < len(vals); i += 2 {
		parts = append(parts, fmt.Sprintf("%s=%s", vals[i], vals[i+1]))
	}

	log.Print(strings.Join(parts, " "))
}

func (api *API) newEvent(eventType string, vals ...any) *Event {
	now := time.Now()

	ev := &Event{
		start: now,
		Time:  now.Format(time.RFC3339Nano),
		Data: map[string]any{
			"type": eventType,
		},
	}

	ev.Set(vals...)

	return ev
}

func (es *eventState) Close() {
	es.mu.Lock()
	defer es.mu.Unlock()

	for _, target := range es.targets {
		close(target.done)
	}
}

func (es *eventState) WriteEvent(ctx context.Context, ev *Event) {
	ev.Set("durationMS", time.Since(ev.start).Milliseconds())

	rnd := rand.Float64() //nolint:gosec

	es.mu.Lock()
	defer es.mu.Unlock()

	for _, hook := range es.hooks {
		hook(ctx, ev)
	}

	es.updateEventRates(ev)
	eventsPerSecond := es.successEventsPerSecond + es.failureEventsPerSecond

	for _, target := range es.targets {
		// TODO: Reserve some portion for errors if we're over
		// TODO: Separate rates for successes, errors, logs
		prob := target.eventsPerSecond / eventsPerSecond

		if rnd < prob {
			ev2 := *ev
			ev2.SampleRate = int64(math.Round(1 / prob))
			target.events = append(target.events, &ev2)
		}
	}
}

func (es *eventState) updateEventRates(ev *Event) {
	bucket := &es.successEventsPerSecond
	if ev.Data["responseError"] != nil {
		bucket = &es.failureEventsPerSecond
	}

	*bucket++

	now := time.Now()

	if !es.lastEvent.IsZero() {
		*bucket /= (now.Sub(es.lastEvent).Seconds() + 1)
	}

	es.lastEvent = now
}

func (es *eventState) flushLoop(target *eventTarget) {
	t := time.NewTicker(time.Duration(target.writePeriodSeconds * float64(time.Second)))
	defer t.Stop()

	for {
		select {
		case <-target.done:
			es.flush(target)
			return

		case <-t.C:
			es.flush(target)
		}
	}
}

func (es *eventState) flush(target *eventTarget) {
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

func (ev *Event) Set(vals ...any) {
	if len(vals)%2 != 0 {
		panic(vals)
	}

	for i := 0; i < len(vals); i += 2 {
		ev.Data[vals[i].(string)] = vals[i+1]
	}
}

func EventHookBuildInfo(_ context.Context, ev *Event) {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		panic("ReadBuildInfo() failed")
	}

	ev.Set(
		"goVersion", buildInfo.GoVersion,
		"goPackagePath", buildInfo.Path,
		"goMainModuleVersion", buildInfo.Main.Version,
	)
}

func EventHookSpanID(ctx context.Context, ev *Event) {
	spanID := ctx.Value(ContextSpanID)

	if spanID != nil {
		ev.Set("spanID", spanID.(string))
	}
}

func EventHookMetrics(_ context.Context, ev *Event) {
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
			ev.Set(name, sample.Value.Uint64())
		case metrics.KindFloat64:
			ev.Set(name, sample.Value.Float64())
		}
	}
}

func EventHookRUsage(_ context.Context, ev *Event) {
	rusage := &syscall.Rusage{}

	err := syscall.Getrusage(syscall.RUSAGE_SELF, rusage)
	if err != nil {
		panic(err)
	}

	ev.Set(
		"rUsageUTime", time.Duration(rusage.Utime.Nano()).Seconds(),
		"rUsageSTime", time.Duration(rusage.Stime.Nano()).Seconds(),
	)
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
