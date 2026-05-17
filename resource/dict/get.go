package dict

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/murouse/pgo"
	"github.com/murouse/pgo/resource"
)

type getConfig struct {
	extendBuilder resource.ExtendBuilderFunc
}

// Get возвращает одну активную запись по её ID (где deleted_at IS NULL).
// Если запись не найдена, возвращает r.cfg.NotFoundErr.
func (r *Resource[TM, TID]) Get(ctx context.Context, id TID, columns []string, opts ...GetOption) (TM, error) {
	cfg := buildGetConfig(opts)
	var zero TM

	qb := pgo.Sq().
		Select(columns...).
		From(r.cfg.Table).
		Where(sq.Eq{
			"id":         id,
			"deleted_at": nil,
		})

	qb = cfg.extendBuilder(qb)

	var item TM
	if err := r.db.Get(ctx, qb, &item); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return zero, r.cfg.NotFoundErr
		}
		return zero, fmt.Errorf("get: %w", err)
	}

	return item, nil
}

func buildGetConfig(opts []GetOption) *getConfig {
	cfg := &getConfig{
		extendBuilder: func(b sq.SelectBuilder) sq.SelectBuilder { return b },
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

type GetOption func(cfg *getConfig)

// WithGetExtendBuilder позволяет кастомизировать SQL-запрос (например, добавить JOIN или блокировку FOR SHARE)
func WithGetExtendBuilder(extendBuilder resource.ExtendBuilderFunc) GetOption {
	return func(cfg *getConfig) {
		cfg.extendBuilder = extendBuilder
	}
}
