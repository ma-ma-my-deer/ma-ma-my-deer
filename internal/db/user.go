package db

import (
	"context"
	"database/sql"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// User represents a user in the database
type UserTable struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID        uuid.UUID    `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	Email     string       `bun:"email,notnull,unique" json:"email"`
	Password  string       `bun:"password,notnull" json:"password"`
	Name      string       `bun:"name,notnull" json:"name"`
	CreatedAt sql.NullTime `bun:"created_at,default:current_timestamp" json:"created_at"`
	UpdatedAt sql.NullTime `bun:"updated_at,default:current_timestamp" json:"updated_at"`
}

// CreateUserParams contains the parameters for creating a user
type CreateUserParams struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// CreateUserRow represents the returned data from creating a user (without password)
type CreateUserRow struct {
	ID        uuid.UUID    `json:"id"`
	Email     string       `json:"email"`
	Name      string       `json:"name"`
	CreatedAt sql.NullTime `json:"created_at"`
	UpdatedAt sql.NullTime `json:"updated_at"`
}

// CreateUser creates a new user in the database
func (d *DB) CreateUser(ctx context.Context, arg CreateUserParams) (CreateUserRow, error) {
	user := &User{
		Email:    arg.Email,
		Password: arg.Password,
		Name:     arg.Name,
	}

	_, err := d.db.NewInsert().Model(user).Exec(ctx)
	if err != nil {
		return CreateUserRow{}, errors.Wrap(err, "failed to create user")
	}

	return CreateUserRow{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

// GetUserByEmail returns a user by email
func (d *DB) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := d.db.NewSelect().
		Model(&user).
		Where("email = ?", email).
		Scan(ctx)

	if err != nil {
		if err == sql.ErrNoRows {
			return User{}, errors.Wrapf(err, "user not found by email: %s", email)
		}
		return User{}, errors.Wrapf(err, "failed to get user by email: %s", email)
	}

	return user, nil
}
