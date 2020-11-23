package cluster

import (
	"context"

	clnt "github.com/ipfs/ipfs-cluster/api/rest/client"
	ma "github.com/multiformats/go-multiaddr"
)

// New creates an instance of cluster client.
func New(ctx context.Context, addr string) (clnt.Client, error) {
	maAddr, err := ma.NewMultiaddr(addr)
	if err != nil {
		return nil, err
	}
	return clnt.NewDefaultClient(&clnt.Config{
		APIAddr: maAddr,
	})
}
