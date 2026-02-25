package repository

import "context"

// Writer defines the interface for writing pollen data to storage.
type Writer interface {
	SaveRaw(ctx context.Context, snapshot PollenSnapshot) error
	UpdateCache(ctx context.Context, locationID string, snapshot PollenSnapshot) error
}
