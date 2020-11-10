package drive

import (
	"context"
	"testing"

	"github.com/meowdada/ipfstor/options"
)

func TestDrive(t *testing.T) {
	ctx := context.Background()

	d, err := DirectOpen("testDrive", options.OpenDrive().SetDirectory("/home/jack/Desktop/testDrive"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close(ctx)
}
