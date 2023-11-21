package reactive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReactiveCycleBlocking(t *testing.T) {
	obs := New[int](1)
	sub := obs.Subscribe()
	defer sub.Cancel()
	go obs.Publish(1)
	v := <-sub.Channel()
	assert.Equal(t, 1, v)
}

func TestReactiveCycleNonBlocking(t *testing.T) {
	obs := New[int](2)
	sub := obs.Subscribe()
	defer sub.Cancel()
	obs.Publish(1)
	v := <-sub.Channel()
	assert.Equal(t, 1, v)
}

func TestReactiveCycleNonBlockingMultiple(t *testing.T) {
	obs := New[int](2)
	sub := obs.Subscribe()
	defer sub.Cancel()
	obs.Publish(1)
	obs.Publish(2)
	v := <-sub.Channel()
	assert.Equal(t, 1, v)
	v = <-sub.Channel()
	assert.Equal(t, 2, v)
}

func TestReactiveCycleNonBlockingMultipleSubscribers(t *testing.T) {
	obs := New[int](2)
	sub1 := obs.Subscribe()
	defer sub1.Cancel()
	sub2 := obs.Subscribe()
	defer sub2.Cancel()
	obs.Publish(1)
	obs.Publish(2)
	v := <-sub1.Channel()
	assert.Equal(t, 1, v)
	v = <-sub2.Channel()
	assert.Equal(t, 1, v)
	v = <-sub1.Channel()
	assert.Equal(t, 2, v)
	v = <-sub2.Channel()
	assert.Equal(t, 2, v)
}

func TestReactiveCycleNonBlockingMultipleSubscribersCancel(t *testing.T) {
	obs := New[int](2)
	sub1 := obs.Subscribe()
	sub2 := obs.Subscribe()
	sub1.Cancel()
	obs.Publish(1)
	obs.Publish(2)
	v := <-sub2.Channel()
	assert.Equal(t, 1, v)
	v = <-sub2.Channel()
	assert.Equal(t, 2, v)

	v = <-sub1.Channel()
	assert.Equal(t, 0, v) // zero value means channel is closed
}

func TestReactiveCycleLoop(t *testing.T) {
	obs := New[int](100)
	go func() {
		for i := 0; i < 100; i++ {
			obs.Publish(i)
		}
	}()
	sub1 := obs.Subscribe()
	c1 := sub1.Channel()
	defer sub1.Cancel()
	sub2 := obs.Subscribe()
	c2 := sub2.Channel()
	defer sub2.Cancel()
	sub3 := obs.Subscribe()
	c3 := sub3.Channel()
	defer sub3.Cancel()

	for i := 0; i < 100; i++ {
		v := <-c1
		assert.Equal(t, i, v)
		v = <-c2
		assert.Equal(t, i, v)
		v = <-c3
		assert.Equal(t, i, v)

	}
}

func FuzzTestDataIntegrity(f *testing.F) {
	obs := New[string](100)

	sub1 := obs.Subscribe()
	c1 := sub1.Channel()
	defer sub1.Cancel()
	sub2 := obs.Subscribe()
	c2 := sub2.Channel()
	defer sub2.Cancel()
	sub3 := obs.Subscribe()
	c3 := sub3.Channel()
	defer sub3.Cancel()

	for _, v := range []string{"a", "b", "c", "d", "e", "f", "1", "2", "12a", "p45", "123412", "09322", "qwerty", "asdfgh", "zxcvbn"} {
		f.Add(v)
	}

	f.Fuzz(func(t *testing.T, a string) {
		obs.Publish(a)
		v := <-c1
		assert.Equal(t, a, v)
		v = <-c2
		assert.Equal(t, a, v)
		v = <-c3
		assert.Equal(t, a, v)

	})
}
