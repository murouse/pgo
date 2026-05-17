package dict

import (
	"context"
	"fmt"

	"github.com/murouse/pgo"
	"github.com/murouse/pgo/resource"
)

type (
	afterCreateFunc[TID any] func(ctx context.Context, id TID) error
)

type createConfig[TID any] struct {
	preCheck    resource.QueryFunc
	afterCreate afterCreateFunc[TID]
}

// Create вставляет новую запись в таблицу и возвращает её ID.
func (r *Resource[TM, TID]) Create(ctx context.Context, data map[string]any, opts ...CreateOption[TID]) (TID, error) {
	cfg := buildCreateConfig[TID](opts)
	var zero TID

	qb := pgo.Sq().
		Insert(r.cfg.Table).
		SetMap(data).
		Suffix("RETURNING id")

	return pgo.GetInTx(ctx, r.db, func(ctx context.Context) (TID, error) {
		if err := cfg.preCheck(ctx); err != nil {
			return zero, fmt.Errorf("pre check: %w", err)
		}

		var id TID
		if err := r.db.Get(ctx, qb, &id); err != nil {
			if cn, ok := pgo.IsDuplicateKeyError(err); ok && cn == r.cfg.UniqueIndex {
				return zero, r.cfg.AlreadyExistErr
			}
			return zero, fmt.Errorf("insert: %w", err)
		}

		if err := cfg.afterCreate(ctx, id); err != nil {
			return zero, fmt.Errorf("after create: %w", err)
		}

		return id, nil
	})
}

func buildCreateConfig[TID any](opts []CreateOption[TID]) *createConfig[TID] {
	cfg := &createConfig[TID]{
		preCheck:    func(_ context.Context) error { return nil },
		afterCreate: func(_ context.Context, _ TID) error { return nil },
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

type CreateOption[TID any] func(cfg *createConfig[TID])

func WithPreCheckCreate[TID any](preCheck resource.QueryFunc) CreateOption[TID] {
	return func(cfg *createConfig[TID]) {
		cfg.preCheck = preCheck
	}
}

func WithAfterCreate[TID any](afterCreate afterCreateFunc[TID]) CreateOption[TID] {
	return func(cfg *createConfig[TID]) {
		cfg.afterCreate = afterCreate
	}
}
