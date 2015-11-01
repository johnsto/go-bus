package bus

import (
	"sync"
)

type PublishFlag int

const (
	// Async causes each handler to be triggered in a separate Goroutine
	Async PublishFlag = 1 << 0
)

// Handler is called whenever a value is sent on a particular topic.
type Handler interface {
	// On is called each time a value is received on a particular topic.
	On(b *Bus, t, v interface{})
}

// HandlerFunc is an adaptor that allows a handler function to act as a
// Handler itself.
type HandlerFunc func(b *Bus, t, v interface{})

func (h HandlerFunc) On(b *Bus, t, v interface{}) {
	h(b, t, v)
}

// UnsubscribeFunc unsubscribes a handler.
type UnsubscribeFunc func() bool

var defaultBus *Bus
var once sync.Once

// getDefaultBus returns the default bus, creating it if necessary.
func getDefaultBus() *Bus {
	once.Do(func() {
		defaultBus = NewBus()
	})
	return defaultBus
}

// Bus is an in-memory event bus that simplifies communication between
// otherwise distinct components. A bus contains a number of topics, each
// of which has a number of handlers. When a value is published onto a topic,
// each of that topic's handlers are called with that value.
type Bus struct {
	lock   sync.RWMutex
	topics map[interface{}][]Handler
}

// NewBus creates and returns a new Bus.
func NewBus() *Bus {
	return &Bus{
		topics: make(map[interface{}][]Handler),
	}
}

// Subscribe causes the passed Handler to be called when data is published
// to the named topic on this Bus. It returns a function that can be called to
// unsubscribe the handler.
func (b *Bus) Subscribe(t interface{}, h Handler) UnsubscribeFunc {
	b.lock.Lock()
	defer b.lock.Unlock()

	// Add handler to topic, creating topic if not there already
	if hs, ok := b.topics[t]; !ok {
		b.topics[t] = []Handler{h}
	} else {
		b.topics[t] = append(hs, h)
	}

	// Unsubscribe function
	return func() bool {
		return b.Unsubscribe(t, h)
	}
}

// SubscribeFunc registers the handler function on the given topic, returning
// a function that can be called to deregister itself.
func (b *Bus) SubscribeFunc(t interface{}, h func(b *Bus, t, v interface{})) UnsubscribeFunc {
	hf := HandlerFunc(h)
	return b.Subscribe(t, &hf)
}

// Unsubscribe removes the specified handler from the given topic on this Bus,
// returning true on success (i.e. the handler was found and removed)
func (b *Bus) Unsubscribe(t interface{}, h Handler) bool {
	b.lock.Lock()
	defer b.lock.Unlock()

	// Find and remove handler from topic
	a := b.topics[t]
	for i, h2 := range a {
		if h2 == h {
			b.topics[t] = append(a[:i], a[i+1:]...)

			// Remove topic if no handlers are subscribed to it
			if len(b.topics[t]) == 0 {
				delete(b.topics, t)
			}

			return true
		}
	}

	return false
}

// Publish sends the given value to all handlers subscribed to the named
// topic on this Bus. If the `Async` flag is passed, this function will call
// each handler in a separate goroutine and return without blocking.
func (b *Bus) Publish(t interface{}, v interface{}, flags ...PublishFlag) (int, error) {
	var f PublishFlag = 0
	for _, flag := range flags {
		f = f | flag
	}

	b.lock.RLock()
	hs := b.topics[t]
	b.lock.RUnlock()

	if f&Async != 0 {
		for _, h := range hs {
			go h.On(b, t, v)
		}
		return len(hs), nil
	}

	for _, h := range hs {
		h.On(b, t, v)
	}
	return len(hs), nil
}

// PublishAll sends the given value to all handlers registered on all topics
// on this Bus. Each handler is called in its own goroutine, so this
// function will return immediately with the number of handlers called.
// If the same Handler is registered on multiple topics or buses, the handler
// will be called multiple times.
func (b *Bus) PublishAll(v interface{}) (int, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	c := 0
	for t, hs := range b.topics {
		for _, h := range hs {
			go h.On(b, t, v)
		}
		c += len(hs)
	}

	return c, nil
}

// Subscribe causes the passed Handler to be called when data is published
// to the named topic on the default Bus. It returns a function that can be
// called to unsubscribe the handler.
func Subscribe(t interface{}, h Handler) UnsubscribeFunc {
	return getDefaultBus().Subscribe(t, h)
}

// SubscribeFunc registers the handler function on the given topic, returning
// a function that can be called to deregister itself.
func SubscribeFunc(t interface{}, h func(b *Bus, t, v interface{})) UnsubscribeFunc {
	return getDefaultBus().SubscribeFunc(t, h)
}

// Publish sends the given value to all handlers subscribed to the named
// topic on the default Bus.
func Publish(t interface{}, v interface{}, flags ...PublishFlag) (int, error) {
	return getDefaultBus().Publish(t, v, flags...)
}

// Unsubscribe removes the specified handler from the given topic on the
// default Bus, returning true on success (i.e. the handler was found and
// removed)
func Unsubscribe(t interface{}, h Handler) {
	getDefaultBus().Unsubscribe(t, h)
}
