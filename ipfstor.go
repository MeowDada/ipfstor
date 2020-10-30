package ipfstor

import (
	"context"
	"io"

	"github.com/ipfs/go-cid"
)

// IterDriveFn is a callback function that applys to each iterated file
// in the driver.
type IterDriveFn func(ctx context.Context, key string, cid cid.Cid, size int64, owner string) error

// ListResult denotes the listing driver result.
type ListResult struct {
	Key   string
	Cid   cid.Cid
	Size  int64
	Owner string
}

// Driver acts as a cloud drive and provides common cloud drive APIs.
type Driver interface {
	List(ctx context.Context) ([]ListResult, error)

	// Iter iterates all available files in this driver. If error occurs while
	// iterating, the whole process will halt and return the iterating error
	// immediately.
	Iter(ctx context.Context, iterCb IterDriveFn) error

	// Add adds a file to the driver with given key.
	Add(ctx context.Context, key string, reader io.Reader) error

	// Get gets a file from the driver by reading from the returned stream reader.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Remove removes a file from the driver.
	Remove(ctx context.Context, key string) error

	// Close closes the driver.
	Close(ctx context.Context) error
}
