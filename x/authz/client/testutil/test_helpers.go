package testutil

import (
	"github.com/onomyprotocol/cosmos-sdk/testutil"
	clitestutil "github.com/onomyprotocol/cosmos-sdk/testutil/cli"
	"github.com/onomyprotocol/cosmos-sdk/testutil/network"
	"github.com/onomyprotocol/cosmos-sdk/x/authz/client/cli"
)

func ExecGrantAuthorization(val *network.Validator, args []string) (testutil.BufferWriter, error) {
	cmd := cli.NewCmdGrantAuthorization()
	clientCtx := val.ClientCtx
	return clitestutil.ExecTestCLICmd(clientCtx, cmd, args)
}
