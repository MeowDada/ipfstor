package ipfs

import (
	"fmt"

	ipfsClient "github.com/ipfs/go-ipfs-http-client"
	coreiface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/multiformats/go-multiaddr"
)

const (
	// ProtocolIPv4 uses ip address v4.
	ProtocolIPv4 = "ip4"

	// ProtocolIPv6 uses ip address v6.
	ProtocolIPv6 = "ip6"

	// ProtocolIPFS uses ipfs.
	ProtocolIPFS = "ipfs"

	// ProtocolIPNS uses ipns.
	ProtocolIPNS = "ipns"

	// NetworkTCP uses tcp.
	NetworkTCP = "tcp"

	// NetworkUDP uses udp.
	NetworkUDP = "udp"
)

var (
	// DefaultAPIAddress denotes the default api address.
	DefaultAPIAddress = APIAddr{
		Protocol: ProtocolIPv4,
		Addr:     "127.0.0.1",
		Network:  NetworkTCP,
		Port:     5001,
	}
)

// Protocol denotes the protocol to be used.
type Protocol string

// Network denotes using tcp or udp.
type Network string

// APIAddr denotes an address in ipfs.
type APIAddr struct {
	Protocol Protocol
	Addr     string
	Network  Network
	Port     int
}

func (a *APIAddr) String() string {
	return fmt.Sprintf("/%s/%s/%s/%d", a.Protocol, a.Addr, a.Network, a.Port)
}

// NewLocalAPI creates a local ipfs api instance backed by http client.
// An error will raise if there is no local running ipfs daemon.
func NewLocalAPI(addr APIAddr) (coreiface.CoreAPI, error) {
	ma, err := multiaddr.NewMultiaddr(addr.String())
	if err != nil {
		return nil, err
	}
	return ipfsClient.NewApi(ma)
}

// GetLocalAPI fetches all ipfs instances from given APIs.
func GetLocalAPI(addrs []APIAddr) (apis []coreiface.CoreAPI, err error) {
	for i := range addrs {
		ma, err := multiaddr.NewMultiaddr(addrs[i].String())
		if err != nil {
			return nil, err
		}

		api, err := ipfsClient.NewApi(ma)
		if err != nil {
			return nil, err
		}

		apis = append(apis, api)
	}

	return apis, nil
}
