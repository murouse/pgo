package dict

import (
	"github.com/murouse/pgo/resource"
)

// Resource отвечает за crud над справочниками с soft-delete
type Resource[TM, TID any] struct {
	db  resource.DB
	cfg *Config
}

func New[TM, TID any](db resource.DB, cfg *Config) *Resource[TM, TID] {
	return &Resource[TM, TID]{
		db:  db,
		cfg: cfg,
	}
}
