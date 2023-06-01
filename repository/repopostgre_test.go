//go:build integration

package repopository

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	godotenv.Load("../.env")
	user := os.Getenv("POSTGRES_DB_USER")
	passwd := os.Getenv("POSTGRES_DB_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB_NAME")

	db, err := Connect(ctx, fmt.Sprintf("postgres://%s:%s@localhost:5432", user, passwd), dbName)
	assert.Nil(t, err)

	err = db.Ping(ctx)
	assert.Nil(t, err)
}
