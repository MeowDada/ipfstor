package ipfstor

import (
	"context"
	"fmt"
	"testing"

	"github.com/meowdada/ipfstor/pkg/ipfs"
)

func TestUser(t *testing.T) {
	api, err := ipfs.NewLocalAPI()
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	user := NewUser(api)

	k, err := user.Key(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(k)
}
