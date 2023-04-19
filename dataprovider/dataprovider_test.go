package dataprovider

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateValidateSuccess(t *testing.T) {
	address := "somerandomaddressthatisvalid"

	c := New(context.Background(), 1*time.Second)

	d := c.ProvideData(address)
	ok := c.ValidateData(address, d)

	assert.True(t, ok)
}

func TestGenerateValidateFailAddress(t *testing.T) {
	address := "somerandomaddressthatisvalid"

	c := New(context.Background(), 1*time.Second)

	d := c.ProvideData(address)
	ok := c.ValidateData("somerandomaddressthatisnotvalid", d)

	assert.False(t, ok)
}

func TestGenerateValidateFailData(t *testing.T) {
	address := "somerandomaddressthatisvalid"

	c := New(context.Background(), 1*time.Second)

	c.ProvideData(address)
	ok := c.ValidateData(address, []byte{})

	assert.False(t, ok)
}

func TestGenerateValidateFailTimePassed(t *testing.T) {
	address := "somerandomaddressthatisvalid"

	c := New(context.Background(), 100*time.Millisecond)
	d := c.ProvideData(address)

	time.Sleep(200 * time.Millisecond)

	ok := c.ValidateData(address, d)

	assert.False(t, ok)
}
