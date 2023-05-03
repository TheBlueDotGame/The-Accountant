//go:build integration

package repopostgre

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/bartossh/Computantis/repohelper"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	godotenv.Load("../.env")
	user := os.Getenv("MONGO_DB_USER")
	passwd := os.Getenv("MONGO_DB_PASSWORD")
	dbName := os.Getenv("MONGO_DB_NAME")

	cfg := repohelper.DBConfig{
		ConnStr:      fmt.Sprintf("postgres://%s:%s@localhost:5432", user, passwd),
		DatabaseName: dbName,
		Token:        "19130b090d70afb384b6ebcb8572701a974e3a1090947bfc785b980841bfb054",
		TokenExpire:  100,
	}

	db, err := Connect(ctx, cfg)
	assert.Nil(t, err)

	err = db.Ping(ctx)
	assert.Nil(t, err)
}
