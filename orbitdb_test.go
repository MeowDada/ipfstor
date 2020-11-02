package ipfstor

import (
	"context"
	"testing"

	"github.com/meowdada/ipfstor/pkg/ipfs"
)

func TestGet(t *testing.T) {
	ctx := context.Background()

	api, err := ipfs.NewLocalAPI()
	if err != nil {
		t.Fatal(err)
	}

	driver, err := NewDriver(ctx, api, "kvstore")
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close(ctx)

	addr := driver.Address()
	t.Log(addr)

	_, err = driver.Get(ctx, "12512")
	if err != nil {
		t.Error(err)
	}
}
