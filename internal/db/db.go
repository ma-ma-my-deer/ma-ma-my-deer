package db

import (
	"database/sql"

	"github.com/my-deer/mydeer/internal/errors"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

// DB represents the database connection and operations
type DB struct {
	db *bun.DB
}

// New creates a new DB instance with the given sql.DB connection
func New(sqlDB *sql.DB) *DB {
	bunDB := bun.NewDB(sqlDB, pgdialect.New())

	// Register models
	bunDB.RegisterModel((*UserTable)(nil))

	return &DB{
		db: bunDB,
	}
}

// Begin starts a new transaction
func (d *DB) Begin() (bun.Tx, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return bun.Tx{}, errors.Wrap(err, "failed to begin transaction", "Internal error", 500)
	}
	return tx, nil
}
