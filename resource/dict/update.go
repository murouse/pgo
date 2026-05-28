package dict

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/murouse/pgo"
	"github.com/murouse/pgo/resource"
	"github.com/samber/lo"
)

type updateConfig struct {
	preCheck     resource.QueryFunc
	afterUpdate  resource.QueryFunc
	compositeKey map[string]any
}

// Update обновляет поля записи по ID, если новые значения отличаются от существующих.
func (r *Resource[TM, TID]) Update(ctx context.Context, id TID, data map[string]any, opts ...UpdateOption) (bool, error) {
	if len(data) == 0 {
		return false, nil
	}

	cfg := buildUpdateConfig(opts)

	qbExists := pgo.Sq().
		Select("1").
		From(r.cfg.Table).
		Where(sq.Eq{
			"id":         id,
			"deleted_at": nil,
		}).
		Suffix("FOR UPDATE")

	if len(cfg.compositeKey) > 0 {
		qbExists = qbExists.Where(cfg.compositeKey)
	}

	changes := lo.Map(lo.Entries(data), func(e lo.Entry[string, any], _ int) sq.Sqlizer {
		return sq.Expr(fmt.Sprintf("%s IS DISTINCT FROM ?", e.Key), e.Value)
	})

	qbUpdate := pgo.Sq().
		Update(r.cfg.Table).
		SetMap(data).
		Set("updated_at", sq.Expr("now()")).
		Where(sq.Eq{
			"id":         id,
			"deleted_at": nil,
		}).
		Where(sq.Or(changes))

	if len(cfg.compositeKey) > 0 {
		qbUpdate = qbUpdate.Where(cfg.compositeKey)
	}

	return pgo.GetInTx(ctx, r.db, func(ctx context.Context) (bool, error) {
		var dummy TID
		if err := r.db.Get(ctx, qbExists, &dummy); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return false, r.cfg.NotFoundErr
			}
			return false, fmt.Errorf("check exists: %w", err)
		}

		if err := cfg.preCheck(ctx); err != nil {
			return false, fmt.Errorf("pre check: %w", err)
		}

		res, err := r.db.Exec(ctx, qbUpdate)
		if err != nil {
			if cn, ok := pgo.IsDuplicateKeyError(err); ok && cn == r.cfg.UniqueIndex {
				return false, r.cfg.AlreadyExistErr
			}
			return false, fmt.Errorf("update: %w", err)
		}

		if err = cfg.afterUpdate(ctx); err != nil {
			return false, fmt.Errorf("after update: %w", err)
		}

		return res.RowsAffected() > 0, nil
	})
}

func buildUpdateConfig(opts []UpdateOption) *updateConfig {
	cfg := &updateConfig{
		preCheck:     func(_ context.Context) error { return nil },
		afterUpdate:  func(_ context.Context) error { return nil },
		compositeKey: nil,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

type UpdateOption func(cfg *updateConfig)

func WithUpdatePreCheck(preCheck resource.QueryFunc) UpdateOption {
	return func(cfg *updateConfig) {
		cfg.preCheck = preCheck
	}
}

func WithAfterUpdate(afterUpdate resource.QueryFunc) UpdateOption {
	return func(cfg *updateConfig) {
		cfg.afterUpdate = afterUpdate
	}
}

func WithUpdateCompositeKey(compositeKey map[string]any) UpdateOption {
	return func(cfg *updateConfig) {
		cfg.compositeKey = compositeKey
	}
}
