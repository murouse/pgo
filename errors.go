package pgo

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func IsDuplicateKeyError(err error) (constraintName string, ok bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		// duplicate key value violates unique constraint
		return pgErr.ConstraintName, true
	}

	return "", false
}

func IsForeignKeyViolationError(err error) (constraintName string, ok bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		// foreign key violation
		return pgErr.ConstraintName, true
	}

	return "", false
}
