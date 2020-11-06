package ipfstor

import (
	"context"
	"io"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
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

// PeerInfo denotes the infomation about each connected peers.
type PeerInfo struct {
	ID peer.ID
}

// User represents an IPFS user.
type User interface {
	// GenerateKeyFile generates new keys for builing a private IPFS netowrk.
	// If the target file exists, it will be overwrites by a new create one.
	GenerateKeyFile(ctx context.Context, path string) error

	// Key represents the key of the user.
	Key(ctx context.Context) (string, error)

	// AddPeer to the bootstrap list.
	//
	// The address of a peer might look as below:
	// /ip4/172.16.0.113/tcp/4001/ipfs/QmV7Thb3mjuWa1xDK5UrgtG7SSYFt4PSyvo6CjcnA5gZAg
	AddPeer(ctx context.Context, addr string) error

	// GetPeers gets the connected peers of this user.
	GetPeers(ctx context.Context) ([]PeerInfo, error)
}

// Driver acts as a cloud drive and provides common cloud drive APIs.
type Driver interface {
	// Address returns the ipfs path of this driver.
	Address() string

	// List returns a result consist of all existing objects in the driver.
	List(ctx context.Context) ([]ListResult, error)

	// Iter iterates all available files in this driver. If error occurs while
	// iterating, the whole process will halt and return the iterating error
	// immediately.
	Iter(ctx context.Context, iterCb IterDriveFn) error

	// Add adds a file to the driver with given key.
	Add(ctx context.Context, key, path string) error

	// Get gets a file from the driver by reading from the returned stream reader.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Remove removes a file from the driver.
	Remove(ctx context.Context, key string) error

	// Close closes the driver.
	Close(ctx context.Context) error
}
