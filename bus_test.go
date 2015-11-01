package bus

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestEmpty checks the behaviour of a Bus with no listeners
func TestEmpty(t *testing.T) {
	bus := NewBus()
	n, err := bus.Publish("hello", "world")
	assert.NoError(t, err, "broadcasting to empty bus should not fail")
	assert.Equal(t, 0, n, "zero channels should receive message")
}

// TestDefault checks the behaviour of the default Bus
func TestDefaultBus(t *testing.T) {
	n, err := Publish("hello", "world")
	assert.NoError(t, err, "publishing to default bus should not fail")
	assert.Equal(t, 0, n, "zero channels should receive message")

	defer SubscribeFunc("test", func(b *Bus, tp, v interface{}) {
		assert.Equal(t, "test", tp, "only subscriber")
		assert.Equal(t, "hello", v, "only subscriber")
	})()

	n, err = Publish("test", "hello")
	assert.Equal(t, 1, n, "publishing to single subscriber")
	assert.NoError(t, err, "publishing to single subscriber")
}

func TestPublish(t *testing.T) {
	bus := NewBus()
	c1 := 0
	c2 := 0

	dereg1 := bus.SubscribeFunc("test", func(b *Bus, tp, v interface{}) {
		c1++
		assert.Equal(t, "test", tp, "in first subscriber")
		assert.Equal(t, "hello", v, "in first subscriber")
	})

	n, err := bus.Publish("test", "hello")
	assert.Equal(t, 1, n, "publishing to first subscriber")
	assert.Equal(t, 1, c1, "publishing to first subscriber")
	assert.NoError(t, err, "publishing to first subscriber")

	dereg2 := bus.SubscribeFunc("test", func(b *Bus, tp, v interface{}) {
		c2++
		assert.Equal(t, "test", tp, "in second subscriber")
		assert.Equal(t, "hello", v, "in second subscriber")
	})

	n, err = bus.Publish("test", "hello")
	assert.Equal(t, 2, n, "publishing to both subscribers")
	assert.NoError(t, err, "publishing to both subscribers")
	assert.Equal(t, 2, c1, "publishing to first subscriber")
	assert.Equal(t, 1, c2, "publishing to first subscriber")

	assert.True(t, dereg1(), "first handler should unsubscribe")

	n, err = bus.Publish("test", "hello")
	assert.Equal(t, 1, n, "publishing to second subscriber")
	assert.NoError(t, err, "publishing to second subscriber")
	assert.Equal(t, 2, c1, "publishing to first subscriber")
	assert.Equal(t, 2, c2, "publishing to first subscriber")
	assert.True(t, dereg2(), "second handler should unsubscribe")
}

// TestPublishAsync asserts that the `Async` flag does not block `Publish`.
func TestPublishAsync(t *testing.T) {
	c := make(chan int)
	sw := false
	defer SubscribeFunc("test", func(b *Bus, tp, v interface{}) {
		assert.Equal(t, 0, <-c) // wait for signal
		assert.True(t, sw)
		c <- 1
	})()

	n, err := Publish("test", "hello", Async)
	assert.NoError(t, err)
	assert.Equal(t, 1, n)
	sw = true
	c <- 0
	assert.Equal(t, 1, <-c)
}

func TestOnce(t *testing.T) {
	cnt := 0
	defer OnceFunc("test", func(b *Bus, tp, v interface{}) {
		cnt++
		assert.Equal(t, 1, cnt)
	})()

	n, err := Publish("test", "hello")
	assert.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, 1, cnt)

	n, err = Publish("test", "hello")
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, 1, cnt)
}

func TestOnceAync(t *testing.T) {
	c := make(chan int)
	cnt := 0
	defer OnceFunc("test", func(b *Bus, tp, v interface{}) {
		assert.Equal(t, 0, <-c)
		cnt++
		assert.Equal(t, 1, cnt)
		c <- 1
	})()

	n, err := Publish("test", "hello", Async)
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = Publish("test", "hello", Async)
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	c <- 0
	assert.Equal(t, 1, <-c)
	assert.Equal(t, 1, cnt)
}
