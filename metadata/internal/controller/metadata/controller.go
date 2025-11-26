package metadata

import (
	"context"
	"errors"
	"log"

	"movieexample.com/metadata/internal/repository"
	"movieexample.com/metadata/pkg/model"
)

// ErrNotFound is returned when a requested record is not found.
var ErrNotFound = errors.New("not found")

type metadataRepository interface {
	Get(ctx context.Context, id string) (*model.Metadata, error)
	Put(ctx context.Context, id string, metadata *model.Metadata) error
}

// Controller defines a metadata service controller.
type Controller struct {
	repo  metadataRepository
	cache metadataRepository
}

// New creates a metadata service controller.
func New(repo metadataRepository, cache metadataRepository) *Controller {
	return &Controller{repo, cache}
}

// Get returns movie metadata by id.
func (c *Controller) Get(ctx context.Context, id string) (*model.Metadata, error) {
	cacheRes, err := c.cache.Get(ctx, id)
	if err == nil {
		log.Println("Returning metadata from a cache for " + id)
		return cacheRes, nil
	}

	res, err := c.repo.Get(ctx, id)
	if err != nil && errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}

	return res, nil
}
