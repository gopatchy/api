package patchy

import (
	"context"

	"github.com/gopatchy/event"
)

func (api *API) EventClient() *event.Client {
	return api.eventClient
}

func (api *API) Log(ctx context.Context, vals ...any) {
	api.eventClient.Log(ctx, vals...)
}

func (api *API) SetEventData(ctx context.Context, vals ...any) {
	ev := ctx.Value(ContextEvent)

	if ev == nil {
		return
	}

	ev.(*event.Event).Set(vals...)
}

func EventHookSpanID(ctx context.Context, ev *event.Event) {
	spanID := ctx.Value(ContextSpanID)

	if spanID != nil {
		ev.Set("spanID", spanID.(string))
	}
}
