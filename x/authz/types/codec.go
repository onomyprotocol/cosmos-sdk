package types

import (
	types "github.com/onomyprotocol/cosmos-sdk/codec/types"
	sdk "github.com/onomyprotocol/cosmos-sdk/types"
	"github.com/onomyprotocol/cosmos-sdk/types/msgservice"
	"github.com/onomyprotocol/cosmos-sdk/x/authz/exported"
	bank "github.com/onomyprotocol/cosmos-sdk/x/bank/types"
	staking "github.com/onomyprotocol/cosmos-sdk/x/staking/types"
)

// RegisterInterfaces registers the interfaces types with the interface registry
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.MsgRequest)(nil),
		&MsgGrantAuthorizationRequest{},
		&MsgRevokeAuthorizationRequest{},
		&MsgExecAuthorizedRequest{},
	)

	registry.RegisterInterface(
		"cosmos.authz.v1beta1.Authorization",
		(*exported.Authorization)(nil),
		&bank.SendAuthorization{},
		&GenericAuthorization{},
		&staking.StakeAuthorization{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
