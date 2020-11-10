package ipfsutil

import (
	ipfsClient "github.com/ipfs/go-ipfs-http-client"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	ma "github.com/multiformats/go-multiaddr"
)

const (
	// DefaultAPIAddress is the default address of a ipfs http gateway endpoint.
	DefaultAPIAddress = "/ip4/127.0.0.1/tcp/5001"
)

// NewAPI creates an ipfs api instance backed by a http client.
func NewAPI(addr string) (coreiface.CoreAPI, error) {
	maddr, err := ma.NewMultiaddr(addr)
	if err != nil {
		return nil, err
	}
	return ipfsClient.NewApi(maddr)
}
