package params

import (
	"github.com/onomyprotocol/cosmos-sdk/client"
	"github.com/onomyprotocol/cosmos-sdk/codec"
	"github.com/onomyprotocol/cosmos-sdk/codec/types"
)

// EncodingConfig specifies the concrete encoding types to use for a given app.
// This is provided for compatibility between protobuf and amino implementations.
type EncodingConfig struct {
	InterfaceRegistry types.InterfaceRegistry
	Marshaler         codec.Marshaler
	TxConfig          client.TxConfig
	Amino             *codec.LegacyAmino
}
