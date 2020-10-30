package object

import (
	"github.com/ipfs/go-cid"
)

// Metadata denotes the metadata of a file.
type Metadata struct {
	Cid   cid.Cid
	Owner string
	Size  int64
}
