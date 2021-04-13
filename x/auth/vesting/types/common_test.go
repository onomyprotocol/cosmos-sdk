package types_test

import (
	"github.com/onomyprotocol/cosmos-sdk/simapp"
)

var (
	app      = simapp.Setup(false)
	appCodec = simapp.MakeTestEncodingConfig().Marshaler
)
