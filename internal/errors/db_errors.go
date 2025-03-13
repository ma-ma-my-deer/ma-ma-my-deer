package errors

import (
	"database/sql"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/lib/pq"
)

// PostgreSQL error codes
const (
	pqErrUniqueViolation     = "23505"
	pqErrForeignKeyViolation = "23503"
)

// WrapDBError wraps database errors with appropriate application errors
func WrapDBError(err error) error {
	if err == nil {
		return nil
	}

	// Check for common errors
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}

	// Check for PostgreSQL specific errors
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case pqErrUniqueViolation:
			constraint := pqErr.Constraint
			field := extractFieldFromConstraint(constraint)
			return Wrap(err, ErrDBDuplicate, "Duplicate entry", 409).WithDetails(map[string]string{
				"field":      field,
				"constraint": constraint,
			})
		case pqErrForeignKeyViolation:
			return Wrap(err, ErrDBExecution, "Referenced resource not found", 400)
		}
	}

	// Default to a generic database error
	return Wrap(err, ErrDBExecution, "Database operation failed", 500)
}

// extractFieldFromConstraint tries to extract the field name from a constraint name
func extractFieldFromConstraint(constraint string) string {
	// Common pattern in PostgreSQL: table_name_field_name_key
	parts := strings.Split(constraint, "_")
	if len(parts) < 3 {
		return ""
	}

	// Return the second to last part as the field name
	return parts[len(parts)-2]
}
