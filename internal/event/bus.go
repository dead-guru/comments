package event

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"strconv"
	"sync"
	"time"

	"deadcomments/internal/domain"
	"deadcomments/internal/repository"
)

type Handler interface {
	Key() string
	Handle(context.Context, domain.Event) error
}

type Bus struct {
	store    *repository.EventRepository
	mu       sync.RWMutex
	handlers []Handler
}

func NewBus(store *repository.EventRepository) *Bus {
	return &Bus{store: store}
}

func (b *Bus) Subscribe(handler Handler) {
	if handler == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, handler)
}

func (b *Bus) Publish(ctx context.Context, event domain.Event) error {
	if event.ID == "" {
		event.ID = NewID()
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	if event.Payload == nil {
		event.Payload = map[string]any{}
	}
	if event.Type == "" || event.AggregateType == "" || event.AggregateID == "" {
		return errors.New("event type, aggregate type, and aggregate id are required")
	}
	if err := b.store.Store(ctx, event); err != nil {
		return err
	}
	b.mu.RLock()
	handlers := append([]Handler(nil), b.handlers...)
	b.mu.RUnlock()
	for _, handler := range handlers {
		err := handler.Handle(ctx, event)
		if markErr := b.store.MarkDelivery(ctx, event.ID, handler.Key(), err); markErr != nil {
			log.Printf("event delivery mark failed handler=%s event=%s: %v", handler.Key(), event.ID, markErr)
		}
		if err != nil {
			log.Printf("event handler failed handler=%s event=%s type=%s: %v", handler.Key(), event.ID, event.Type, err)
		}
	}
	return nil
}

func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "evt-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return hex.EncodeToString(b[:])
}

type Publisher interface {
	Publish(context.Context, domain.Event) error
}
