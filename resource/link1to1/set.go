package link1to1

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/murouse/pgo"
	"github.com/murouse/pgo/resource"
)

type setConfig struct {
	preCheck resource.QueryFunc
	afterSet resource.QueryFunc
}

// Set устанавливает связь и возвращает true, если связь была добавлена, и false, если связь уже была
func (r *Resource[TID]) Set(ctx context.Context, leftID, rightID TID, opts ...SetOption) (bool, error) {
	cfg := buildSetConfig(opts)

	qb := pgo.Sq().
		Insert(r.cfg.Table).
		SetMap(sq.Eq{
			r.cfg.LeftColumn:  leftID,
			r.cfg.RightColumn: rightID,
		}).
		Suffix(fmt.Sprintf("ON CONFLICT ON CONSTRAINT %s DO NOTHING", r.cfg.UniqueIndex))

	return pgo.GetInTx(ctx, r.db, func(ctx context.Context) (bool, error) {
		if err := cfg.preCheck(ctx); err != nil {
			return false, fmt.Errorf("pre check: %w", err)
		}

		tag, err := r.db.Exec(ctx, qb)
		if err == nil {
			// если запись уже была, выходим с false
			if tag.RowsAffected() == 0 {
				return false, nil
			}

			if err = cfg.afterSet(ctx); err != nil {
				return false, fmt.Errorf("after set: %w", err)
			}

			return true, nil
		}

		if cn, ok := pgo.IsForeignKeyViolationError(err); ok && cn == r.cfg.LeftForeignKey {
			return false, r.cfg.LeftNotFoundErr
		}

		if cn, ok := pgo.IsForeignKeyViolationError(err); ok && cn == r.cfg.RightForeignKey {
			return false, r.cfg.RightNotFoundErr
		}

		return false, fmt.Errorf("insert: %w", err)
	})
}

func buildSetConfig(opts []SetOption) *setConfig {
	cfg := &setConfig{
		preCheck: func(_ context.Context) error { return nil },
		afterSet: func(_ context.Context) error { return nil },
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

type SetOption func(cfg *setConfig)

func WithPreCheckSet(preCheck resource.QueryFunc) SetOption {
	return func(cfg *setConfig) {
		cfg.preCheck = preCheck
	}
}

func WithAfterSet(afterSet resource.QueryFunc) SetOption {
	return func(cfg *setConfig) {
		cfg.afterSet = afterSet
	}
}
