package dict

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/murouse/pgo"
	"github.com/murouse/pgo/resource"
)

type deleteConfig struct {
	preCheck     resource.QueryFunc
	compositeKey map[string]any
}

// Delete помечает запись как удаленную (soft delete) после выполнения проверок preCheck.
func (r *Resource[TM, TID]) Delete(ctx context.Context, id TID, opts ...DeleteOption) error {
	cfg := buildDeleteConfig(opts)

	qb := pgo.Sq().
		Update(r.cfg.Table).
		SetMap(sq.Eq{
			"updated_at": sq.Expr("now()"),
			"deleted_at": sq.Expr("now()"),
		}).
		Where(sq.Eq{
			"id":         id,
			"deleted_at": nil,
		})

	if len(cfg.compositeKey) > 0 {
		qb = qb.Where(cfg.compositeKey)
	}

	return pgo.InTx(ctx, r.db, func(ctx context.Context) error {
		if err := cfg.preCheck(ctx); err != nil {
			return fmt.Errorf("pre check: %w", err)
		}

		tag, err := r.db.Exec(ctx, qb)
		if err != nil {
			return fmt.Errorf("delete: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return r.cfg.NotFoundErr
		}

		return nil
	})
}

func buildDeleteConfig(opts []DeleteOption) *deleteConfig {
	cfg := &deleteConfig{
		preCheck:     func(_ context.Context) error { return nil },
		compositeKey: nil,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

type DeleteOption func(cfg *deleteConfig)

func WithDeletePreCheck(preCheck resource.QueryFunc) DeleteOption {
	return func(cfg *deleteConfig) {
		cfg.preCheck = preCheck
	}
}

func WithDeleteCompositeKey(compositeKey map[string]any) DeleteOption {
	return func(cfg *deleteConfig) {
		cfg.compositeKey = compositeKey
	}
}
