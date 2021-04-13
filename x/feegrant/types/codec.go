package types

import (
	"github.com/onomyprotocol/cosmos-sdk/codec/types"
	sdk "github.com/onomyprotocol/cosmos-sdk/types"
	"github.com/onomyprotocol/cosmos-sdk/types/msgservice"
)

// RegisterInterfaces registers the interfaces types with the interface registry
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.MsgRequest)(nil),
		&MsgGrantFeeAllowance{},
		&MsgRevokeFeeAllowance{},
	)

	registry.RegisterInterface(
		"cosmos.feegrant.v1beta1.FeeAllowanceI",
		(*FeeAllowanceI)(nil),
		&BasicFeeAllowance{},
		&PeriodicFeeAllowance{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
