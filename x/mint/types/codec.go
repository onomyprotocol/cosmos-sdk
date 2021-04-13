package types

import (
	"github.com/onomyprotocol/cosmos-sdk/codec"
	cryptocodec "github.com/onomyprotocol/cosmos-sdk/crypto/codec"
)

var (
	amino = codec.NewLegacyAmino()
)

func init() {
	cryptocodec.RegisterCrypto(amino)
	amino.Seal()
}
