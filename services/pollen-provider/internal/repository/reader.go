package repository

import "context"

type PollenReader interface {
	GetAll(ctx context.Context) ([]CacheDoc, error)
	GetByID(ctx context.Context, id string) (*CacheDoc, error)
}
