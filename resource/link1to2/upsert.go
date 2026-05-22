package link1to2

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/murouse/pgo"
	"github.com/murouse/pgo/resource"
	"github.com/samber/lo"
)

type (
	afterUpsertFunc func(ctx context.Context, isModified bool) error
)

type upsertConfig struct {
	preCheck     resource.QueryFunc
	afterUpsert  afterUpsertFunc
	postgresType string
}

// Upsert атомарно синхронизирует связи «один-ко-многим» для указанной левой сущности (leftID).
// Он заменяет текущий набор связанных правых сущностей на новый список (rightIDs).
//
// Метод выполняет следующие шаги в рамках единой транзакции:
//  1. Блокирует строку левой сущности в LeftTable с помощью "FOR UPDATE" для предотвращения
//     состояний гонки (race conditions) при параллельных запросах на запись.
//  2. Удаляет все существующие связи для leftID из LinkTable и возвращает список старых ID.
//  3. Проверяет переданные rightIDs на существование и валидность (deleted_at IS NULL) в RightTable.
//  4. Вставляет новые связи в LinkTable.
//
// Возвращает:
//   - bool: true, если состояние связей в базе данных изменилось (изменилось количество
//     или состав связанных сущностей); false, если итоговый набор связей остался прежним.
//   - error: ошибку Config.DataIntegrityErr, если левая или какая-либо из правых сущностей
//     не найдены/удалены. Либо системную ошибку работы с базой данных.
func (r *Resource[TID]) Upsert(ctx context.Context, leftID TID, rightIDs []TID, opts ...UpsertOption) (bool, error) {
	cfg := buildUpsertConfig[TID](opts)
	insertedRightIDs := lo.Uniq(rightIDs) // защита от дублей

	// Блокируем родительскую запись
	qbLock := pgo.Sq().
		Select("id").
		From(r.cfg.LeftTable).
		Where(sq.Eq{
			"id":         leftID,
			"deleted_at": nil,
		}).
		Suffix("FOR UPDATE")

	// Очищаем связи
	qbDelete := pgo.Sq().
		Delete(r.cfg.LinkTable).
		Where(sq.Eq{r.cfg.LeftColumnID: leftID}).
		Suffix("RETURNING " + r.cfg.RightColumnID)

	// Вставляем связи
	qInsert := fmt.Sprintf(`
	  INSERT INTO %s (%s, %s)
	  SELECT $1, input.%s
	  FROM unnest($2::%s[]) AS input(%s) -- unnest($2::int[])
	  JOIN %s r ON r.id = input.%s AND r.deleted_at IS NULL
	`, r.cfg.LinkTable, r.cfg.LeftColumnID, r.cfg.RightColumnID,
		r.cfg.RightColumnID,
		cfg.postgresType,
		r.cfg.RightColumnID,
		r.cfg.RightTable,
		r.cfg.RightColumnID,
	)

	return pgo.GetInTx(ctx, r.db, func(ctx context.Context) (bool, error) {
		var dummy TID
		if err := r.db.Get(ctx, qbLock, &dummy); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return false, r.cfg.DataIntegrityErr
			}
			return false, fmt.Errorf("lock: %w", err)
		}

		if err := cfg.preCheck(ctx); err != nil {
			return false, fmt.Errorf("pre check: %w", err)
		}

		var deletedRightIDs []TID
		if err := r.db.Select(ctx, qbDelete, &deletedRightIDs); err != nil {
			return false, fmt.Errorf("delete: %w", err)
		}

		var isModified bool
		if len(insertedRightIDs) == 0 {
			isModified = len(deletedRightIDs) != 0
		} else {
			tag, err := r.db.Exec(ctx, pgo.Sql(qInsert, leftID, insertedRightIDs))
			if err != nil {
				return false, fmt.Errorf("insert: %w", err)
			}
			if tag.RowsAffected() != int64(len(insertedRightIDs)) {
				return false, r.cfg.DataIntegrityErr
			}

			isModified = !lo.ElementsMatch(insertedRightIDs, deletedRightIDs)
		}

		if err := cfg.afterUpsert(ctx, isModified); err != nil {
			return false, fmt.Errorf("after upsert: %w", err)
		}

		return isModified, nil
	})
}

// getPostgresType маппит Go-тип идентификатора в соответствующий тип в PostgreSQL.
func getPostgresType[T any]() string {
	var dummy T
	switch any(dummy).(type) {
	case int64, int:
		return "bigint"
	case int32:
		return "int"
	case string:
		return "text"
	case uuid.UUID:
		return "uuid"
	default:
		return "bigint"
	}
}

func buildUpsertConfig[TID any](opts []UpsertOption) *upsertConfig {
	cfg := &upsertConfig{
		preCheck:     func(_ context.Context) error { return nil },
		afterUpsert:  func(_ context.Context, _ bool) error { return nil },
		postgresType: getPostgresType[TID](),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

type UpsertOption func(cfg *upsertConfig)

func WithPreCheckUnset(preCheck resource.QueryFunc) UpsertOption {
	return func(cfg *upsertConfig) {
		cfg.preCheck = preCheck
	}
}

func WithAfterUnset(afterUnset afterUpsertFunc) UpsertOption {
	return func(cfg *upsertConfig) {
		cfg.afterUpsert = afterUnset
	}
}

func WithPostgresType(postgresType string) UpsertOption {
	return func(cfg *upsertConfig) {
		cfg.postgresType = postgresType
	}
}
