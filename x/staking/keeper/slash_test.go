package keeper_test

import (
	// "cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	// distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	// minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/staking/testutil"
)

// tests Jail, Unjail
func (s *KeeperTestSuite) TestRevocation() {
	ctx, keeper := s.ctx, s.stakingKeeper
	require := s.Require()

	valAddr := sdk.ValAddress(PKs[0].Address().Bytes())
	consAddr := sdk.ConsAddress(PKs[0].Address())
	validator := testutil.NewValidator(s.T(), valAddr, PKs[0])

	// initial state
	require.NoError(keeper.SetValidator(ctx, validator))
	require.NoError(keeper.SetValidatorByConsAddr(ctx, validator))
	val, err := keeper.GetValidator(ctx, valAddr)
	require.NoError(err)
	require.False(val.IsJailed())

	// test jail
	require.NoError(keeper.Jail(ctx, consAddr))
	val, err = keeper.GetValidator(ctx, valAddr)
	require.NoError(err)
	require.True(val.IsJailed())

	// test unjail
	require.NoError(keeper.Unjail(ctx, consAddr))
	val, err = keeper.GetValidator(ctx, valAddr)
	require.NoError(err)
	require.False(val.IsJailed())
}

// tests Slash at a future height (must error)
func (s *KeeperTestSuite) TestSlashAtFutureHeight() {
	ctx, keeper := s.ctx, s.stakingKeeper
	require := s.Require()

	consAddr := sdk.ConsAddress(PKs[0].Address())
	validator := testutil.NewValidator(s.T(), sdk.ValAddress(PKs[0].Address().Bytes()), PKs[0])
	require.NoError(keeper.SetValidator(ctx, validator))
	require.NoError(keeper.SetValidatorByConsAddr(ctx, validator))

	fraction := sdkmath.LegacyNewDecWithPrec(5, 1)
	_, err := keeper.Slash(ctx, consAddr, 1, 10, fraction)
	require.Error(err)
}

// // tests Slash at the current height
// func (s *KeeperTestSuite) TestSlashValidatorAtCurrentHeightWithSlashingProtection() {
// 	// use disr types module name to withdraw the reward without errors and
// 	moduleDelegatorName := distrtypes.ModuleName

// 	ctx, k := s.ctx, s.stakingKeeper
// 	require := s.Require()

// 	k.SetSlashingProtestedModules(func() map[string]struct{} {
// 		return map[string]struct{}{
// 			moduleDelegatorName: {},
// 		}
// 	})
// 	k.SetHooks(types.NewMultiStakingHooks(app.DistrKeeper.Hooks()))

// 	valBondTokens := k.TokensFromConsensusPower(ctx, 10)
// 	valReward := k.TokensFromConsensusPower(ctx, 1).ToLegacyDec()
// 	delBondTokens := k.TokensFromConsensusPower(ctx, 2)
// 	delProtectedBondTokens := k.TokensFromConsensusPower(ctx, 4)
// 	delProtectedExpectedReward := k.TokensFromConsensusPower(ctx, 875).QuoRaw(1000) // 0.875
// 	totalDelegation := valBondTokens.Add(delBondTokens).Add(delProtectedBondTokens)

// 	fraction := math.LegacyNewDecWithPrec(5, 1)

// 	// generate delegator account
// 	delAddr := simapp.AddTestAddrs(app, ctx, 1, delBondTokens)[0]
// 	// generate protected delegator account
// 	err := s.bankKeeper.SendCoinsFromAccountToModule(ctx, simapp.AddTestAddrs(app, ctx, 1, delProtectedBondTokens)[0],
// 		moduleDelegatorName, sdk.NewCoins(sdk.NewCoin(k.BondDenom(ctx), delProtectedBondTokens)))
// 	require.NoError(t, err)
// 	delProtectedAddr := app.AccountKeeper.GetModuleAddress(moduleDelegatorName)

// 	// get already created validator
// 	vaConsAddr := sdk.ConsAddress(PKs[0].Address())
// 	// delegate from normal account
// 	val, found := k.GetValidatorByConsAddr(ctx, vaConsAddr)
// 	require.True(t, found)
// 	// call this function here to init the validator in the distribution module
// 	k.AfterValidatorCreated(ctx, val.GetOperator())

// 	// delegate from normal account
// 	delShares := delegate(t, app, ctx, vaConsAddr, delAddr, delBondTokens)
// 	// delegate from protected account
// 	delegate(t, app, ctx, vaConsAddr, delProtectedAddr, delProtectedBondTokens)

// 	// capture the current bond state
// 	bondedPool := k.GetBondedPool(ctx)
// 	oldBondedPoolBalances := app.BankKeeper.GetAllBalances(ctx, bondedPool.GetAddress())
// 	// end block
// 	applyValidatorSetUpdates(t, ctx, app.StakingKeeper, 1)

// 	// mint coins for the distr module
// 	require.NoError(t, app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, sdk.NewCoins(sdk.NewCoin(k.BondDenom(ctx), valReward.TruncateInt()))))
// 	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToModule(ctx, minttypes.ModuleName, distrtypes.ModuleName, sdk.NewCoins(sdk.NewCoin(k.BondDenom(ctx), valReward.TruncateInt()))))
// 	// add reward to the validator to withdraw by the protected module
// 	app.DistrKeeper.AllocateTokensToValidator(ctx, val, sdk.NewDecCoins(sdk.NewDecCoinFromDec(k.BondDenom(ctx), valReward)))

// 	// get current power
// 	power := k.GetLastValidatorPower(ctx, val.GetOperator())
// 	require.Equal(t, k.TokensToConsensusPower(ctx, totalDelegation), power)

// 	// increase the block number to be able to get the reward
// 	ctx = app.BaseApp.NewContext(false, tmproto.Header{Height: app.LastBlockHeight() + 1})
// 	// now slash based on the current power
// 	k.Slash(ctx, vaConsAddr, ctx.BlockHeight(), power, fraction, types.InfractionEmpty)
// 	// end block
// 	applyValidatorSetUpdates(t, ctx, app.StakingKeeper, 1)

// 	// read updated test
// 	val, found = k.GetValidator(ctx, val.GetOperator())
// 	assert.True(t, found)
// 	// power decreased, the protected delegation was remove from the calculation
// 	// since the module is protected from the slashing
// 	expectedPower := k.TokensToConsensusPower(ctx,
// 		totalDelegation.Sub(delProtectedBondTokens).
// 			ToDec().Mul(fraction).TruncateInt())
// 	power = val.GetConsensusPower(k.PowerReduction(ctx))
// 	require.Equal(t, expectedPower, power)

// 	// pool bonded shares decreased
// 	newBondedPoolBalances := app.BankKeeper.GetAllBalances(ctx, bondedPool.GetAddress())
// 	diffTokens := oldBondedPoolBalances.Sub(newBondedPoolBalances).AmountOf(k.BondDenom(ctx))
// 	require.Equal(t, totalDelegation.Sub(delProtectedBondTokens).ToDec().Mul(fraction).TruncateInt().
// 		// add undelegated tokens
// 		Add(delProtectedBondTokens).String(), diffTokens.String())

// 	// check the delegation slashing
// 	unbondDelegationAmount, err := k.Unbond(ctx, delAddr, val.GetOperator(), delShares)
// 	assert.NoError(t, err)
// 	// the amount 50% less because of the slashing
// 	assert.Equal(t, delBondTokens.ToDec().Mul(fraction).TruncateInt(), unbondDelegationAmount)

// 	// check that protected module has no delegation now
// 	_, found = k.GetDelegation(ctx, delProtectedAddr, val.GetOperator())
// 	assert.False(t, found)

// 	delProtectedBalance := app.BankKeeper.GetAllBalances(ctx, delProtectedAddr)
// 	assert.Equal(t, sdk.NewCoins(sdk.NewCoin(k.BondDenom(ctx), delProtectedBondTokens.Add(delProtectedExpectedReward))), delProtectedBalance)
// }

// func delegate(t *testing.T, app *simapp.SimApp, ctx sdk.Context, vaConsAddr sdk.ConsAddress, delAddr sdk.AccAddress, delTokens sdk.Int) sdk.Dec {
// 	t.Helper()

// 	val, found := k.GetValidatorByConsAddr(ctx, vaConsAddr)
// 	require.True(t, found)
// 	delShares, err := k.Delegate(ctx, delAddr, delTokens, types.Unbonded, val, true)
// 	require.NoError(t, err)
// 	require.Equal(t, delTokens.String(), delShares.TruncateInt().String())
// 	return delShares
// }
