package link1to2

import "github.com/murouse/pgo/resource"

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
