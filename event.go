package patchy

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
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

// TODO: Switch to opentelemetry protocol
// TODO: Split event publisher out to its own library
// TODO: Add protocol-level tests

type EventTarget struct {
	client             *resty.Client
	writePeriodSeconds float64
	windowSeconds      float64
	rateClasses        []*eventRateClass
	done               chan bool
	lastEvent          time.Time
	events             []*Event
}

type Event struct {
	start time.Time

	Time       string         `json:"time"`
	SampleRate int64          `json:"samplerate"`
	Data       map[string]any `json:"data"`
}

type EventHook func(context.Context, *Event)

type eventState struct {
	targets         []*EventTarget
	hooks           []EventHook
	tlsClientConfig *tls.Config

	mu sync.Mutex
}

type eventRateClass struct {
	grantRate float64
	criteria  map[string]any

	eventRate float64
}

func (api *API) SetEventTLSClientConfig(config *tls.Config) {
	api.eventState.tlsClientConfig = config
}

func (api *API) AddEventTarget(url string, headers map[string]string, writePeriodSeconds float64) *EventTarget {
	target := &EventTarget{
		client:             resty.New().SetBaseURL(url).SetHeaders(headers),
		writePeriodSeconds: writePeriodSeconds,
		windowSeconds:      100.0,
		done:               make(chan bool),
		lastEvent:          time.Now(),
	}

	if api.eventState.tlsClientConfig != nil {
		target.client.SetTLSClientConfig(api.eventState.tlsClientConfig)
	}

	go api.eventState.flushLoop(target)

	api.eventState.mu.Lock()
	defer api.eventState.mu.Unlock()

	api.eventState.targets = append(api.eventState.targets, target)

	return target
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

func (api *API) Log(ctx context.Context, vals ...any) {
	ev := api.eventState.newEvent("log", vals...)
	api.eventState.writeEvent(ctx, ev)

	parts := []string{}

	for i := 0; i < len(vals); i += 2 {
		parts = append(parts, fmt.Sprintf("%s=%s", vals[i], vals[i+1]))
	}

	log.Print(strings.Join(parts, " "))
}

func (es *eventState) newEvent(eventType string, vals ...any) *Event {
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

func (es *eventState) close() {
	es.mu.Lock()
	defer es.mu.Unlock()

	for _, target := range es.targets {
		close(target.done)
	}
}

func (es *eventState) writeEvent(ctx context.Context, ev *Event) {
	ev.Set("durationMS", time.Since(ev.start).Milliseconds())

	es.mu.Lock()
	defer es.mu.Unlock()

	for _, hook := range es.hooks {
		hook(ctx, ev)
	}

	for _, target := range es.targets {
		target.writeEvent(ev)
	}
}

func (es *eventState) flushLoop(target *EventTarget) {
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

func (es *eventState) flush(target *EventTarget) {
	es.mu.Lock()
	events := target.events
	target.events = nil
	es.mu.Unlock()

	if len(events) == 0 {
		return
	}

	buf := &bytes.Buffer{}
	g := gzip.NewWriter(buf)
	enc := json.NewEncoder(g)

	err := enc.Encode(events)
	if err != nil {
		panic(err)
	}

	err = g.Close()
	if err != nil {
		panic(err)
	}

	resp, err := target.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(buf).
		Post("")
	if err != nil {
		log.Printf("failed write to event target: %s", err)
		return
	}

	if resp.IsError() {
		log.Printf("failed write to event target: %d %s: %s", resp.StatusCode(), resp.Status(), resp.String())
		return
	}
}

func (target *EventTarget) AddRateClass(grantRate float64, vals ...any) {
	if len(vals)%2 != 0 {
		panic(vals)
	}

	erc := &eventRateClass{
		grantRate: grantRate * target.windowSeconds,
		criteria:  map[string]any{},
	}

	for i := 0; i < len(vals); i += 2 {
		erc.criteria[vals[i].(string)] = vals[i+1]
	}

	target.rateClasses = append(target.rateClasses, erc)
}

func (target *EventTarget) writeEvent(ev *Event) {
	now := time.Now()
	secondsSinceLastEvent := now.Sub(target.lastEvent).Seconds()
	target.lastEvent = now

	// Example:
	//   windowSeconds = 100
	//   secondsSinceLastEvent = 25
	//   eventRateMultiplier = 0.75
	eventRateMultiplier := (target.windowSeconds - secondsSinceLastEvent) / target.windowSeconds

	maxProb := 0.0

	for _, erc := range target.rateClasses {
		if !erc.match(ev) {
			continue
		}

		erc.eventRate++
		erc.eventRate *= eventRateMultiplier

		classProb := erc.grantRate / erc.eventRate
		maxProb = math.Max(maxProb, classProb)
	}

	if maxProb <= 0.0 || rand.Float64() > maxProb { //nolint:gosec
		return
	}

	ev2 := *ev
	ev2.SampleRate = int64(math.Max(math.Round(1.0/maxProb), 1.0))
	target.events = append(target.events, &ev2)
}

func (ev *Event) Set(vals ...any) {
	if len(vals)%2 != 0 {
		panic(vals)
	}

	for i := 0; i < len(vals); i += 2 {
		ev.Data[vals[i].(string)] = vals[i+1]
	}
}

func (erc *eventRateClass) match(ev *Event) bool {
	for k, v := range erc.criteria {
		if ev.Data[k] != v {
			return false
		}
	}

	return true
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
