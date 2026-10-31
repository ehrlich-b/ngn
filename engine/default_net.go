package engine

import (
	"bytes"
	_ "embed"
)

// DefaultNetName identifies the embedded NGN-trained network in UCI output.
const DefaultNetName = "<embedded>"

// DefaultNetSHA256 is the digest of the embedded network file.
const DefaultNetSHA256 = "812152293dad04184599b2f9ab82779168e6e134b221de136d8f8daae86d9f22"

//go:embed nets/ngn-h512-81215229.nnue
var defaultNetBytes []byte

// DefaultNetBytes returns a copy of the embedded network file.
func DefaultNetBytes() []byte { return append([]byte(nil), defaultNetBytes...) }

// LoadDefaultNet decodes the embedded network with the same validation as EvalFile.
func LoadDefaultNet() (*NGNN1Network, error) {
	return ReadNGNN1(bytes.NewReader(defaultNetBytes))
}
