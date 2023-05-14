package repository

import "context"

// RunMigration satisfies the RepositoryProvider interface
// as PostgreSQL migrations are run on when database is created in docker-compose-postgresql.yml.
func (DataBase) RunMigration(_ context.Context) error {
	return nil
}
