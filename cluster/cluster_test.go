package cluster

import (
	"context"
	"testing"

	"github.com/meowdada/ipfstor/options"
	"github.com/stretchr/testify/require"
)

func TestCluster(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := NewConfig(ctx, options.Cluster().
		SetConfigPath("service.json").
		SetIdentityPath("identity.json"),
	)
	if err != nil {
		require.NoError(t, err)
	}

	cls, err := New(ctx, "service.json", "identity.json", nil)
	require.NoError(t, err)

	cls.Start()
	cls.Stop(ctx)
}
