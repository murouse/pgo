package link1to1

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/murouse/pgo"
	"github.com/murouse/pgo/resource"
)

type unsetConfig struct {
	preCheck   resource.QueryFunc
	afterUnset resource.QueryFunc
}

// Unset удаляет связь и возвращает true, если связь была удалена, и false, если связи не существовало
func (r *Resource[TID]) Unset(ctx context.Context, leftID, rightID TID, opts ...UnsetOption) (bool, error) {
	cfg := buildUnsetConfig(opts)

	qb := pgo.Sq().
		Delete(r.cfg.Table).
		Where(sq.Eq{
			r.cfg.LeftColumn:  leftID,
			r.cfg.RightColumn: rightID,
		})

	return pgo.GetInTx(ctx, r.db, func(ctx context.Context) (bool, error) {
		if err := cfg.preCheck(ctx); err != nil {
			return false, fmt.Errorf("pre check: %w", err)
		}

		tag, err := r.db.Exec(ctx, qb)
		if err != nil {
			return false, fmt.Errorf("delete: %w", err)
		}

		// если связи не было, выходим с false
		if tag.RowsAffected() == 0 {
			return false, nil
		}

		if err = cfg.afterUnset(ctx); err != nil {
			return false, fmt.Errorf("after unset: %w", err)
		}

		return true, nil
	})
}

func buildUnsetConfig(opts []UnsetOption) *unsetConfig {
	cfg := &unsetConfig{
		preCheck:   func(_ context.Context) error { return nil },
		afterUnset: func(_ context.Context) error { return nil },
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

type UnsetOption func(cfg *unsetConfig)

func WithPreCheckUnset(preCheck resource.QueryFunc) UnsetOption {
	return func(cfg *unsetConfig) {
		cfg.preCheck = preCheck
	}
}

func WithAfterUnset(afterUnset resource.QueryFunc) UnsetOption {
	return func(cfg *unsetConfig) {
		cfg.afterUnset = afterUnset
	}
}
