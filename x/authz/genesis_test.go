package authz_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/onomyprotocol/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/onomyprotocol/cosmos-sdk/simapp"
	sdk "github.com/onomyprotocol/cosmos-sdk/types"
	authz "github.com/onomyprotocol/cosmos-sdk/x/authz"
	"github.com/onomyprotocol/cosmos-sdk/x/authz/keeper"
	bank "github.com/onomyprotocol/cosmos-sdk/x/bank/types"
)

type GenesisTestSuite struct {
	suite.Suite

	ctx    sdk.Context
	keeper keeper.Keeper
}

func (suite *GenesisTestSuite) SetupTest() {
	checkTx := false
	app := simapp.Setup(checkTx)

	suite.ctx = app.BaseApp.NewContext(checkTx, tmproto.Header{Height: 1})
	suite.keeper = app.AuthzKeeper
}

var (
	granteePub  = secp256k1.GenPrivKey().PubKey()
	granterPub  = secp256k1.GenPrivKey().PubKey()
	granteeAddr = sdk.AccAddress(granteePub.Address())
	granterAddr = sdk.AccAddress(granterPub.Address())
)

func (suite *GenesisTestSuite) TestImportExportGenesis() {
	coins := sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1_000)))

	now := suite.ctx.BlockHeader().Time
	grant := &bank.SendAuthorization{SpendLimit: coins}
	err := suite.keeper.Grant(suite.ctx, granteeAddr, granterAddr, grant, now.Add(time.Hour))
	suite.Require().NoError(err)
	genesis := authz.ExportGenesis(suite.ctx, suite.keeper)

	// Clear keeper
	suite.keeper.Revoke(suite.ctx, granteeAddr, granterAddr, grant.MethodName())

	authz.InitGenesis(suite.ctx, suite.keeper, genesis)
	newGenesis := authz.ExportGenesis(suite.ctx, suite.keeper)
	suite.Require().Equal(genesis, newGenesis)
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}
