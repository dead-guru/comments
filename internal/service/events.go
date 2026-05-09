package service

import (
	"context"
	"strconv"

	"deadcomments/internal/actionctx"
	"deadcomments/internal/domain"
	"deadcomments/internal/event"
)

func publish(ctx context.Context, publisher event.Publisher, evt domain.Event) error {
	if publisher == nil {
		return nil
	}
	if evt.ActorAdminID == nil {
		evt.ActorAdminID = actionctx.AdminID(ctx)
	}
	return publisher.Publish(ctx, evt)
}

func optionalPublisher(events []event.Publisher) event.Publisher {
	if len(events) == 0 {
		return nil
	}
	return events[0]
}

func int64Ptr(v int64) *int64 {
	return &v
}

func stringPtr(v string) *string {
	return &v
}

func int64ID(v int64) string {
	return strconv.FormatInt(v, 10)
}
