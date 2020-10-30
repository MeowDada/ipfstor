package ipfs

import (
	ipfsClient "github.com/ipfs/go-ipfs-http-client"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
)

// NewLocalAPI creates a local ipfs api instance backed by http client.
// An error will raise if there is no local running ipfs daemon.
func NewLocalAPI() (coreiface.CoreAPI, error) {
	return ipfsClient.NewLocalApi()
}
