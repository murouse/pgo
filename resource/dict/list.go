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
	tableAlias    string
	orderBy       []string
}

// List возвращает список активных записей (где deleted_at IS NULL).
//
// Если используется опция WithTableAlias, вызывающий код обязан
// самостоятельно указывать заданный алиас в качестве префикса для всех передаваемых
// колонок в аргументе columns и внутри опции WithOrderBy.
func (r *Resource[TM, TID]) List(ctx context.Context, columns []string, opts ...ListOption) ([]TM, error) {
	cfg := buildListConfig(opts)

	columnPrefix := ""
	from := r.cfg.Table
	if cfg.tableAlias != "" {
		columnPrefix = cfg.tableAlias + "."
		from = r.cfg.Table + " AS " + cfg.tableAlias
	}

	orderBy := []string{columnPrefix + "id"}
	if len(cfg.orderBy) > 0 {
		orderBy = cfg.orderBy
	}

	qb := pgo.Sq().
		Select(columns...).
		From(from).
		Where(sq.Eq{
			columnPrefix + "deleted_at": nil,
		}).
		OrderBy(orderBy...)

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
		tableAlias:    "",
		orderBy:       nil,
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

// WithTableAlias задает алиас для целевой таблицы (например, "FROM tobacco AS t").
//
// Использование алиаса накладывает на вызывающий код обязанность вручную прописывать
// этот префикс во всех запрашиваемых колонках и условиях сортировки.
// Пример:
//
//	repo.List(ctx, []string{"t.id", "t.name"}, dict.WithTableAlias("t"), dict.WithOrderBy("t.name ASC"))
func WithTableAlias(tableAlias string) ListOption {
	return func(cfg *listConfig) {
		cfg.tableAlias = tableAlias
	}
}

// WithOrderBy задает кастомную сортировку.
func WithOrderBy(orderBy ...string) ListOption {
	return func(cfg *listConfig) {
		cfg.orderBy = orderBy
	}
}
