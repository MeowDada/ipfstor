package options

const (
	defaultClusterConfigPath   = "service.json"
	defaultClusterIdentityPath = "identity.json"
)

// ClusterOption configures a cluster option.
type ClusterOption struct {
	ConfigPath   *string
	IdentityPath *string
	SrcURL       *string
	PeerAddrs    []string
	Secret       *string
	RandomPorts  *bool
}

// SetConfigPath sets the config path of this option.
func (o *ClusterOption) SetConfigPath(path string) *ClusterOption {
	o.ConfigPath = strPtr(path)
	return o
}

// SetIdentityPath sets the identity path of this option.
func (o *ClusterOption) SetIdentityPath(path string) *ClusterOption {
	o.IdentityPath = strPtr(path)
	return o
}

// SetSourceURL sets the source url of this option.
func (o *ClusterOption) SetSourceURL(url string) *ClusterOption {
	o.SrcURL = &url
	return o
}

// SetPeerAddr appends peer address to the option.
func (o *ClusterOption) SetPeerAddr(addrs ...string) *ClusterOption {
	o.PeerAddrs = append(o.PeerAddrs, addrs...)
	return o
}

// SetSecret sets the secret key of the cluster option.
func (o *ClusterOption) SetSecret(secret string) *ClusterOption {
	if len(secret) == 0 {
		return o
	}
	o.Secret = strPtr(secret)
	return o
}

// SetRandomPorts sets the flag of random port option.
func (o *ClusterOption) SetRandomPorts(flag bool) *ClusterOption {
	o.RandomPorts = boolPtr(flag)
	return o
}

// Cluster creates an empty Cluster option.
func Cluster() *ClusterOption {
	return &ClusterOption{}
}

// DefaultCluster creates a default Cluster option.
func DefaultCluster() *ClusterOption {
	return &ClusterOption{
		ConfigPath:   strPtr(defaultClusterConfigPath),
		IdentityPath: strPtr(defaultClusterIdentityPath),
		RandomPorts:  boolPtr(false),
	}
}

// MergeClusterOptions merges multiple cluster option into a single one.
func MergeClusterOptions(opts ...*ClusterOption) *ClusterOption {
	opt := Cluster()
	for i := range opts {
		if opts[i] == nil {
			continue
		}
		if opts[i].ConfigPath != nil {
			opt.ConfigPath = opts[i].ConfigPath
		}
		if opts[i].IdentityPath != nil {
			opt.IdentityPath = opts[i].IdentityPath
		}
		if opts[i].PeerAddrs != nil {
			opt.PeerAddrs = opts[i].PeerAddrs
		}
		if opts[i].SrcURL != nil {
			opt.SrcURL = opts[i].SrcURL
		}
		if opts[i].Secret != nil {
			opt.Secret = opts[i].Secret
		}
		if opts[i].RandomPorts != nil {
			opt.RandomPorts = opts[i].RandomPorts
		}
	}
	return opt
}
