//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := DBConfig{
		ConnStr:      "postgres://computantis:computantis@localhost:5432",
		DatabaseName: "computantis",
		IsSSL:        false,
	}

	db, err := Connect(ctx, cfg)
	assert.Nil(t, err)

	err = db.Ping(ctx)
	assert.Nil(t, err)
}
