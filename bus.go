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
func (b *Bus) Subscribe(topic interface{}, h Handler) UnsubscribeFunc {
	b.lock.Lock()
	defer b.lock.Unlock()

	// Add handler to topic, creating topic if not there already
	if hs, ok := b.topics[topic]; !ok {
		b.topics[topic] = []Handler{h}
	} else {
		b.topics[topic] = append(hs, h)
	}

	// Unsubscribe function
	return func() bool {
		return b.Unsubscribe(topic, h)
	}
}

// SubscribeFunc registers the handler function on the given topic, returning
// a function that can be called to deregister itself.
func (b *Bus) SubscribeFunc(topic interface{}, h func(b *Bus, t, v interface{})) UnsubscribeFunc {
	hf := HandlerFunc(h)
	return b.Subscribe(topic, &hf)
}

// Unsubscribe removes the specified handler from the given topic on this Bus,
// returning true on success (i.e. the handler was found and removed)
func (b *Bus) Unsubscribe(topic interface{}, h Handler) bool {
	b.lock.Lock()
	defer b.lock.Unlock()

	// Find and remove handler from topic
	a := b.topics[topic]
	for i, h2 := range a {
		if h2 == h {
			b.topics[topic] = append(a[:i], a[i+1:]...)

			// Remove topic if no handlers are subscribed to it
			if len(b.topics[topic]) == 0 {
				delete(b.topics, topic)
			}

			return true
		}
	}

	return false
}

// Publish sends the given value to all handlers subscribed to the named
// topic on this Bus. If the `Async` flag is passed, this function will call
// each handler in a separate goroutine and return without blocking.
func (b *Bus) Publish(topic interface{}, value interface{}, flags ...PublishFlag) (int, error) {
	var f PublishFlag = 0
	for _, flag := range flags {
		f = f | flag
	}

	b.lock.RLock()
	hs := b.topics[topic]
	b.lock.RUnlock()

	if f&Async != 0 {
		// Call each handler in a separate Goroutine
		for _, h := range hs {
			go h.On(b, topic, value)
		}
		return len(hs), nil
	}

	for _, h := range hs {
		h.On(b, topic, value)
	}
	return len(hs), nil
}

// PublishAll sends the given value to all handlers registered on all topics
// on this Bus. If the same Handler is registered on multiple topics or buses,
// the handler will be called multiple times. Returns the number of handlers
// fired.
func (b *Bus) PublishAll(value interface{}) (int, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	c := 0
	for t, hs := range b.topics {
		for _, h := range hs {
			go h.On(b, t, value)
		}
		c += len(hs)
	}

	return c, nil
}

// Subscribe causes the passed Handler to be called when data is published
// to the named topic on the default Bus. It returns a function that can be
// called to unsubscribe the handler.
func Subscribe(topic interface{}, h Handler) UnsubscribeFunc {
	return getDefaultBus().Subscribe(topic, h)
}

// SubscribeFunc registers the handler function on the given topic, returning
// a function that can be called to deregister itself.
func SubscribeFunc(topic interface{}, fn func(b *Bus, t, v interface{})) UnsubscribeFunc {
	return getDefaultBus().SubscribeFunc(topic, fn)
}

// Publish sends the given value to all handlers subscribed to the named
// topic on the default Bus.
func Publish(topic interface{}, value interface{}, flags ...PublishFlag) (int, error) {
	return getDefaultBus().Publish(topic, value, flags...)
}

// Unsubscribe removes the specified handler from the given topic on the
// default Bus, returning true on success (i.e. the handler was found and
// removed)
func Unsubscribe(topic interface{}, h Handler) {
	getDefaultBus().Unsubscribe(topic, h)
}
