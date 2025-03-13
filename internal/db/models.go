package db

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// User represents a user in the database
type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID        uuid.UUID    `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	Email     string       `bun:"email,notnull,unique" json:"email"`
	Password  string       `bun:"password,notnull" json:"password"`
	Name      string       `bun:"name,notnull" json:"name"`
	CreatedAt sql.NullTime `bun:"created_at,default:current_timestamp" json:"created_at"`
	UpdatedAt sql.NullTime `bun:"updated_at,default:current_timestamp" json:"updated_at"`
}
