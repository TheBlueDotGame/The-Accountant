package mongorepo

import (
	"context"
	"errors"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type markerUp struct {
	Name string `json:"name"`
}

type migration struct {
	run  func(ctx context.Context, user *mongo.Database) error
	name string
}

// Migration describes migration that is made in the repository database.
type Migration struct {
	Name string `json:"name" bson:"name"`
}

var migrations = []migration{
	{
		name: "index_name_migrations",
		run: func(ctx context.Context, user *mongo.Database) error {
			_, err := user.Collection(migrationsCollection).
				Indexes().
				CreateOne(ctx, mongo.IndexModel{
					Keys: bson.M{
						"name": 1,
					},
					Options: options.Index().SetUnique(true),
				})
			return err
		},
	},
	{
		name: "index_public_key_addresses",
		run: func(ctx context.Context, user *mongo.Database) error {
			_, err := user.Collection(addressesCollection).
				Indexes().
				CreateOne(ctx, mongo.IndexModel{
					Keys: bson.M{
						"public_key": 1,
					},
					Options: options.Index().SetUnique(true),
				})
			if err != nil {
				return err
			}
			_, err = user.Collection(addressesCollection).
				Indexes().
				CreateOne(ctx, mongo.IndexModel{
					Keys: bson.M{
						"public_key": "text",
					},
				})
			return err
		},
	},
	{
		name: "index_hash_transactions_permanent",
		run: func(ctx context.Context, user *mongo.Database) error {
			_, err := user.Collection(transactionsPermanentCollection).
				Indexes().
				CreateOne(ctx, mongo.IndexModel{
					Keys: bson.M{
						"hash": 1,
					},
					Options: options.Index().SetUnique(true),
				})
			return err
		},
	},
	{
		name: "index_hash_transactions_temporary",
		run: func(ctx context.Context, user *mongo.Database) error {
			_, err := user.Collection(transactionsTemporaryCollection).
				Indexes().
				CreateOne(ctx, mongo.IndexModel{
					Keys: bson.M{
						"hash": 1,
					},
					Options: options.Index().SetUnique(true),
				})
			return err
		},
	},
	{
		name: "index_hash_transactions_awaiting",
		run: func(ctx context.Context, user *mongo.Database) error {
			_, err := user.Collection(transactionsAwaitingReceiverCollection).
				Indexes().
				CreateOne(ctx, mongo.IndexModel{
					Keys: bson.M{
						"transaction_hash": 1,
					},
					Options: options.Index().SetUnique(true),
				})
			if err != nil {
				return err
			}
			_, err = user.Collection(transactionsAwaitingReceiverCollection).
				Indexes().
				CreateOne(ctx, mongo.IndexModel{
					Keys: bson.M{
						"receiver_address": 1,
					},
					Options: options.Index().SetUnique(false),
				})
			if err != nil {
				return err
			}
			_, err = user.Collection(transactionsAwaitingReceiverCollection).
				Indexes().
				CreateOne(ctx, mongo.IndexModel{
					Keys: bson.M{
						"issuer_address": 1,
					},
					Options: options.Index().SetUnique(false),
				})
			return err
		},
	},
	{
		name: "index_hash_prev_hash_index_blocks",
		run: func(ctx context.Context, user *mongo.Database) error {
			_, err := user.Collection(blocksCollection).
				Indexes().
				CreateOne(ctx, mongo.IndexModel{
					Keys: bson.M{
						"hash": 1,
					},
					Options: options.Index().SetUnique(true),
				})
			if err != nil {
				return err
			}
			_, err = user.Collection(blocksCollection).
				Indexes().
				CreateOne(ctx, mongo.IndexModel{
					Keys: bson.M{
						"prev_hash": 1,
					},
					Options: options.Index().SetUnique(true),
				})
			if err != nil {
				return err
			}
			_, err = user.Collection(blocksCollection).
				Indexes().
				CreateOne(ctx, mongo.IndexModel{
					Keys: bson.M{
						"index": -1,
					},
					Options: options.Index().SetUnique(true),
				})
			return err
		},
	},
	{
		name: "index_transaction_in_block",
		run: func(ctx context.Context, user *mongo.Database) error {
			_, err := user.Collection(transactionsInBlockCollection).
				Indexes().
				CreateOne(ctx, mongo.IndexModel{
					Keys: bson.M{
						"transaction_hash": 1,
					},
					Options: options.Index().SetUnique(true),
				})
			return err
		},
	},
	{
		name: "index_token",
		run: func(ctx context.Context, user *mongo.Database) error {
			_, err := user.Collection(tokensCollection).
				Indexes().
				CreateOne(ctx, mongo.IndexModel{
					Keys: bson.M{
						"token": 1,
					},
					Options: options.Index().SetUnique(true),
				})
			return err
		},
	},
	{
		name: "index_logger_level_created_at",
		run: func(ctx context.Context, user *mongo.Database) error {
			_, err := user.Collection(tokensCollection).
				Indexes().
				CreateOne(ctx, mongo.IndexModel{
					Keys: bson.M{
						"level": 1,
					},
					Options: options.Index().SetUnique(false),
				})
			if err != nil {
				return err
			}
			_, err = user.Collection(tokensCollection).
				Indexes().
				CreateOne(ctx, mongo.IndexModel{
					Keys: bson.M{
						"created_at": 1,
					},
					Options: options.Index().SetUnique(false),
				})
			return err
		},
	},
	{
		name: "index_validator_status_index",
		run: func(ctx context.Context, user *mongo.Database) error {
			_, err := user.Collection(validatorStatusCollection).
				Indexes().
				CreateOne(ctx, mongo.IndexModel{
					Keys: bson.M{
						"index": -1,
					},
					Options: options.Index().SetUnique(true),
				})
			return err
		},
	},
}

func (c DataBase) migrate(ctx context.Context, migrationsCollection []migration) ([]string, error) {
	migrated := make([]string, 0, len(migrationsCollection))
	var err error
	for _, migration := range migrationsCollection {
		ok, errC := c.checkExists(ctx, migration.name)
		if errC != nil {
			err = errC
			break
		}
		if ok {
			continue
		}
		errN := migration.run(ctx, &c.inner)
		if errN != nil {
			err = errN
			break
		}
		log.Printf("migrated: %s\n", migration.name)
		migrated = append(migrated, migration.name)
	}
	return migrated, err
}

func (c DataBase) checkExists(ctx context.Context, name string) (bool, error) {
	coll := c.inner.Collection(migrationsCollection)
	query := bson.M{"name": name}

	m := markerUp{}
	if err := coll.FindOne(ctx, query).Decode(&m); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, fmt.Errorf("failed to execute find query, %w", err)
	}
	return true, nil
}

func (c DataBase) saveMigrated(ctx context.Context, migrationNames []string) error {
	coll := c.inner.Collection(migrationsCollection)

	documents := make([]interface{}, 0, len(migrationNames))

	for _, name := range migrationNames {
		documents = append(documents, &markerUp{Name: name})
	}

	if _, err := coll.InsertMany(ctx, documents); err != nil {
		return fmt.Errorf("cannot save migrations marker up, %w", err)
	}
	return nil
}

// RunMigrationUp runs all the migrations
func (c DataBase) RunMigration(ctx context.Context) error {
	migrated, err := c.migrate(ctx, migrations)
	if err != nil {
		return err
	}

	if len(migrated) == 0 {
		fmt.Println("Noting to migrate.")
		return nil
	}

	if err := c.saveMigrated(ctx, migrated); err != nil {
		return err
	}

	return nil
}
