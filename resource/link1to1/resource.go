package link1to1

import (
	"github.com/murouse/pgo/resource"
)

// Resource отвечает за проставление/снятие связей в таблицах-связках многие ко многим hard delete
type Resource[TID comparable] struct {
	db  resource.DB
	cfg *Config
}

func New[TID comparable](db resource.DB, cfg *Config) *Resource[TID] {
	return &Resource[TID]{
		db:  db,
		cfg: cfg,
	}
}
