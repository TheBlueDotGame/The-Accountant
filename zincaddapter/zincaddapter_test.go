//go:build integration

package zincaddapter

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

const token = "Basic YWRtaW46emluY3NlYXJjaA==" // Update token before testin.

func TestZincsearchLoggingIntegration(t *testing.T) {
	cfg := Config{"http://localhost:4080", "test_logging", token}
	writer, err := New(cfg)
	assert.Nil(t, err)

	l := log.New(&writer, "TESTING: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Llongfile)

	l.Println("This should to show in the Zincsearch backend. Go and look there to finally verify the test.")
}
