package dict

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/murouse/pgo"
	"github.com/murouse/pgo/resource"
)

type listConfig struct {
	extendBuilder resource.ExtendBuilderFunc
}

// List возвращает список активных записей (где deleted_at IS NULL).
func (r *Resource[TM, TID]) List(ctx context.Context, columns []string, opts ...ListOption) ([]TM, error) {
	cfg := buildListConfig(opts)

	qb := pgo.Sq().
		Select(columns...).
		From(r.cfg.Table).
		Where(sq.Eq{
			"deleted_at": nil,
		}).
		OrderBy("id")

	qb = cfg.extendBuilder(qb)

	var items []TM
	if err := r.db.Select(ctx, qb, &items); err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	return items, nil
}

func buildListConfig(opts []ListOption) *listConfig {
	cfg := &listConfig{
		extendBuilder: func(b sq.SelectBuilder) sq.SelectBuilder { return b },
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

type ListOption func(cfg *listConfig)

func WithListExtendBuilder(extendBuilder resource.ExtendBuilderFunc) ListOption {
	return func(cfg *listConfig) {
		cfg.extendBuilder = extendBuilder
	}
}
