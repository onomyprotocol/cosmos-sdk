package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/cosmos-sdk/x/staking/teststaking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

func TestCalculateRewardsBasic(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})
	tstaking := teststaking.NewHelper(t, ctx, app.StakingKeeper)

	addr := simapp.AddTestAddrs(app, ctx, 2, sdk.NewInt(1000))
	valAddrs := simapp.ConvertAddrsToValAddrs(addr)

	// create validator with 50% commission
	tstaking.Commission = stakingtypes.NewCommissionRates(sdk.NewDecWithPrec(5, 1), sdk.NewDecWithPrec(5, 1), sdk.NewDec(0))
	tstaking.CreateValidator(valAddrs[0], valConsPk1, sdk.NewInt(100), true)

	// end block to bond validator and start new block
	staking.EndBlocker(ctx, app.StakingKeeper)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	tstaking.Ctx = ctx

	// fetch validator and delegation
	val := app.StakingKeeper.Validator(ctx, valAddrs[0])
	del := app.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddrs[0]), valAddrs[0])

	// historical count should be 2 (once for validator init, once for delegation init)
	require.Equal(t, uint64(2), app.DistrKeeper.GetValidatorHistoricalReferenceCount(ctx))

	// end period
	endingPeriod := app.DistrKeeper.IncrementValidatorPeriod(ctx, val)

	// historical count should be 2 still
	require.Equal(t, uint64(2), app.DistrKeeper.GetValidatorHistoricalReferenceCount(ctx))

	// calculate delegation rewards
	rewards := app.DistrKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)

	// rewards should be zero
	require.True(t, rewards.IsZero())

	// allocate some rewards
	initial := int64(10)
	tokens := sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: sdk.NewDec(initial)}}
	app.DistrKeeper.AllocateTokensToValidator(ctx, val, tokens)

	// end period
	endingPeriod = app.DistrKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards
	rewards = app.DistrKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)

	// rewards should be half the tokens
	require.Equal(t, sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: sdk.NewDec(initial / 2)}}, rewards)

	// commission should be the other half
	require.Equal(t, sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: sdk.NewDec(initial / 2)}}, app.DistrKeeper.GetValidatorAccumulatedCommission(ctx, valAddrs[0]).Commission)
}

func TestCalculateRewardsAfterSlash(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	addr := simapp.AddTestAddrs(app, ctx, 2, sdk.NewInt(100000000))
	valAddrs := simapp.ConvertAddrsToValAddrs(addr)
	tstaking := teststaking.NewHelper(t, ctx, app.StakingKeeper)

	// create validator with 50% commission
	tstaking.Commission = stakingtypes.NewCommissionRates(sdk.NewDecWithPrec(5, 1), sdk.NewDecWithPrec(5, 1), sdk.NewDec(0))
	valPower := int64(100)
	tstaking.CreateValidatorWithValPower(valAddrs[0], valConsPk1, valPower, true)

	// end block to bond validator
	staking.EndBlocker(ctx, app.StakingKeeper)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val := app.StakingKeeper.Validator(ctx, valAddrs[0])
	del := app.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddrs[0]), valAddrs[0])

	// end period
	endingPeriod := app.DistrKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards
	rewards := app.DistrKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)

	// rewards should be zero
	require.True(t, rewards.IsZero())

	// start out block height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// slash the validator by 50%
	app.StakingKeeper.Slash(ctx, valConsAddr1, ctx.BlockHeight(), valPower, sdk.NewDecWithPrec(5, 1))

	// retrieve validator
	val = app.StakingKeeper.Validator(ctx, valAddrs[0])

	// increase block height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// allocate some rewards
	initial := app.StakingKeeper.TokensFromConsensusPower(ctx, 10)
	tokens := sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: initial.ToDec()}}
	app.DistrKeeper.AllocateTokensToValidator(ctx, val, tokens)

	// end period
	endingPeriod = app.DistrKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards
	rewards = app.DistrKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)

	// rewards should be half the tokens
	require.Equal(t, sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: initial.QuoRaw(2).ToDec()}}, rewards)

	// commission should be the other half
	require.Equal(t, sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: initial.QuoRaw(2).ToDec()}},
		app.DistrKeeper.GetValidatorAccumulatedCommission(ctx, valAddrs[0]).Commission)
}

func TestCalculateRewardsAfterManySlashes(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	tstaking := teststaking.NewHelper(t, ctx, app.StakingKeeper)
	addr := simapp.AddTestAddrs(app, ctx, 2, sdk.NewInt(100000000))
	valAddrs := simapp.ConvertAddrsToValAddrs(addr)

	// create validator with 50% commission
	valPower := int64(100)
	tstaking.Commission = stakingtypes.NewCommissionRates(sdk.NewDecWithPrec(5, 1), sdk.NewDecWithPrec(5, 1), sdk.NewDec(0))
	tstaking.CreateValidatorWithValPower(valAddrs[0], valConsPk1, valPower, true)

	// end block to bond validator
	staking.EndBlocker(ctx, app.StakingKeeper)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val := app.StakingKeeper.Validator(ctx, valAddrs[0])
	del := app.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddrs[0]), valAddrs[0])

	// end period
	endingPeriod := app.DistrKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards
	rewards := app.DistrKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)

	// rewards should be zero
	require.True(t, rewards.IsZero())

	// start out block height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// slash the validator by 50%
	app.StakingKeeper.Slash(ctx, valConsAddr1, ctx.BlockHeight(), valPower, sdk.NewDecWithPrec(5, 1))

	// fetch the validator again
	val = app.StakingKeeper.Validator(ctx, valAddrs[0])

	// increase block height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// allocate some rewards
	initial := app.StakingKeeper.TokensFromConsensusPower(ctx, 10)
	tokens := sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: initial.ToDec()}}
	app.DistrKeeper.AllocateTokensToValidator(ctx, val, tokens)

	// slash the validator by 50% again
	app.StakingKeeper.Slash(ctx, valConsAddr1, ctx.BlockHeight(), valPower/2, sdk.NewDecWithPrec(5, 1))

	// fetch the validator again
	val = app.StakingKeeper.Validator(ctx, valAddrs[0])

	// increase block height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// allocate some more rewards
	app.DistrKeeper.AllocateTokensToValidator(ctx, val, tokens)

	// end period
	endingPeriod = app.DistrKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards
	rewards = app.DistrKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)

	// rewards should be half the tokens
	require.Equal(t, sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: initial.ToDec()}}, rewards)

	// commission should be the other half
	require.Equal(t, sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: initial.ToDec()}},
		app.DistrKeeper.GetValidatorAccumulatedCommission(ctx, valAddrs[0]).Commission)
}

func TestCalculateRewardsMultiDelegator(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	tstaking := teststaking.NewHelper(t, ctx, app.StakingKeeper)
	addr := simapp.AddTestAddrs(app, ctx, 2, sdk.NewInt(100000000))
	valAddrs := simapp.ConvertAddrsToValAddrs(addr)

	// create validator with 50% commission
	tstaking.Commission = stakingtypes.NewCommissionRates(sdk.NewDecWithPrec(5, 1), sdk.NewDecWithPrec(5, 1), sdk.NewDec(0))
	tstaking.CreateValidator(valAddrs[0], valConsPk1, sdk.NewInt(100), true)

	// end block to bond validator
	staking.EndBlocker(ctx, app.StakingKeeper)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val := app.StakingKeeper.Validator(ctx, valAddrs[0])
	del1 := app.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddrs[0]), valAddrs[0])

	// allocate some rewards
	initial := int64(20)
	tokens := sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: sdk.NewDec(initial)}}
	app.DistrKeeper.AllocateTokensToValidator(ctx, val, tokens)

	// second delegation
	tstaking.Ctx = ctx
	tstaking.Delegate(sdk.AccAddress(valAddrs[1]), valAddrs[0], sdk.NewInt(100))
	del2 := app.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddrs[1]), valAddrs[0])

	// fetch updated validator
	val = app.StakingKeeper.Validator(ctx, valAddrs[0])

	// end block
	staking.EndBlocker(ctx, app.StakingKeeper)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// allocate some more rewards
	app.DistrKeeper.AllocateTokensToValidator(ctx, val, tokens)

	// end period
	endingPeriod := app.DistrKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards for del1
	rewards := app.DistrKeeper.CalculateDelegationRewards(ctx, val, del1, endingPeriod)

	// rewards for del1 should be 3/4 initial
	require.Equal(t, sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: sdk.NewDec(initial * 3 / 4)}}, rewards)

	// calculate delegation rewards for del2
	rewards = app.DistrKeeper.CalculateDelegationRewards(ctx, val, del2, endingPeriod)

	// rewards for del2 should be 1/4 initial
	require.Equal(t, sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: sdk.NewDec(initial * 1 / 4)}}, rewards)

	// commission should be equal to initial (50% twice)
	require.Equal(t, sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: sdk.NewDec(initial)}}, app.DistrKeeper.GetValidatorAccumulatedCommission(ctx, valAddrs[0]).Commission)
}

func TestWithdrawDelegationRewardsBasic(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	balancePower := int64(1000)
	balanceTokens := app.StakingKeeper.TokensFromConsensusPower(ctx, balancePower)
	addr := simapp.AddTestAddrs(app, ctx, 1, sdk.NewInt(1000000000))
	valAddrs := simapp.ConvertAddrsToValAddrs(addr)
	tstaking := teststaking.NewHelper(t, ctx, app.StakingKeeper)

	// set module account coins
	distrAcc := app.DistrKeeper.GetDistributionAccount(ctx)
	require.NoError(t, simapp.FundModuleAccount(app.BankKeeper, ctx, distrAcc.GetName(), sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, balanceTokens))))
	app.AccountKeeper.SetModuleAccount(ctx, distrAcc)

	// create validator with 50% commission
	power := int64(100)
	tstaking.Commission = stakingtypes.NewCommissionRates(sdk.NewDecWithPrec(5, 1), sdk.NewDecWithPrec(5, 1), sdk.NewDec(0))
	valTokens := tstaking.CreateValidatorWithValPower(valAddrs[0], valConsPk1, power, true)

	// assert correct initial balance
	expTokens := balanceTokens.Sub(valTokens)
	require.Equal(t,
		sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, expTokens)},
		app.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(valAddrs[0])),
	)

	// end block to bond validator
	staking.EndBlocker(ctx, app.StakingKeeper)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val := app.StakingKeeper.Validator(ctx, valAddrs[0])

	// allocate some rewards
	initial := app.StakingKeeper.TokensFromConsensusPower(ctx, 10)
	tokens := sdk.DecCoins{sdk.NewDecCoin(sdk.DefaultBondDenom, initial)}

	app.DistrKeeper.AllocateTokensToValidator(ctx, val, tokens)

	// historical count should be 2 (initial + latest for delegation)
	require.Equal(t, uint64(2), app.DistrKeeper.GetValidatorHistoricalReferenceCount(ctx))

	// withdraw rewards
	_, err := app.DistrKeeper.WithdrawDelegationRewards(ctx, sdk.AccAddress(valAddrs[0]), valAddrs[0])
	require.Nil(t, err)

	// historical count should still be 2 (added one record, cleared one)
	require.Equal(t, uint64(2), app.DistrKeeper.GetValidatorHistoricalReferenceCount(ctx))

	// assert correct balance
	exp := balanceTokens.Sub(valTokens).Add(initial.QuoRaw(2))
	require.Equal(t,
		sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, exp)},
		app.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(valAddrs[0])),
	)

	// withdraw commission
	_, err = app.DistrKeeper.WithdrawValidatorCommission(ctx, valAddrs[0])
	require.Nil(t, err)

	// assert correct balance
	exp = balanceTokens.Sub(valTokens).Add(initial)
	require.Equal(t,
		sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, exp)},
		app.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(valAddrs[0])),
	)
}

func TestWithdrawDelegationRewardsVesting(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	val1BalAmt := app.StakingKeeper.TokensFromConsensusPower(ctx, 1000)
	val1VestingAmt := val1BalAmt.QuoRaw(4)
	val1VestingRate := sdk.OneDec().Sub(val1VestingAmt.ToDec().Quo(val1BalAmt.ToDec())) // 25% locked in vesting
	val1DelAmt := app.StakingKeeper.TokensFromConsensusPower(ctx, 100)
	val1Commission := sdk.NewDecWithPrec(5, 1) // 50%

	del1BalAmt := app.StakingKeeper.TokensFromConsensusPower(ctx, 1000)
	del1VestingAmt := del1BalAmt                                                        // all coins vested
	del1DelAmt := del1BalAmt                                                            // full delegation
	del1VestingRate := sdk.OneDec().Sub(del1VestingAmt.ToDec().Quo(del1BalAmt.ToDec())) // 0% locked in vesting

	del2BalAmt := app.StakingKeeper.TokensFromConsensusPower(ctx, 1000)
	del2VestingAmt := sdk.ZeroInt()                                                     // zero in vesting
	del2DelAmt := del2BalAmt                                                            // full delegation
	del2VestingRate := sdk.OneDec().Sub(del2VestingAmt.ToDec().Quo(del2BalAmt.ToDec())) // 100% locked in vesting

	allocatedReward := app.StakingKeeper.TokensFromConsensusPower(ctx, 200)
	allocatedRewardWithoutCommission := allocatedReward.ToDec().Mul(val1Commission)

	delTotalAmt := val1DelAmt.Add(del1DelAmt).Add(del2DelAmt)

	val1FullReward := val1DelAmt.ToDec().Quo(delTotalAmt.ToDec()).Mul(allocatedRewardWithoutCommission)
	del1FullReward := del1DelAmt.ToDec().Quo(delTotalAmt.ToDec()).Mul(allocatedRewardWithoutCommission)
	del2FullReward := del2DelAmt.ToDec().Quo(delTotalAmt.ToDec()).Mul(allocatedRewardWithoutCommission)

	val1Addr := simapp.ConvertAddrsToValAddrs(simapp.AddTestVestingAddrs(app, ctx, 1, val1BalAmt, val1VestingAmt))[0]
	val2Addr := simapp.ConvertAddrsToValAddrs(simapp.AddTestAddrs(app, ctx, 1, val1BalAmt))[0] // just have more than one validator
	del1Addr := simapp.AddTestVestingAddrs(app, ctx, 1, del1BalAmt, del1VestingAmt)[0]
	del2Addr := simapp.AddTestVestingAddrs(app, ctx, 1, del2BalAmt, del2VestingAmt)[0]

	tstaking := teststaking.NewHelper(t, ctx, app.StakingKeeper)

	// set module account coins
	distrAcc := app.DistrKeeper.GetDistributionAccount(ctx)
	require.NoError(t, simapp.FundModuleAccount(app.BankKeeper, ctx, distrAcc.GetName(), sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, allocatedReward))))
	app.AccountKeeper.SetModuleAccount(ctx, distrAcc)

	// create validator with 50% commission
	tstaking.Commission = stakingtypes.NewCommissionRates(val1Commission, val1Commission, sdk.NewDec(0))
	tstaking.CreateValidator(val1Addr, valConsPk1, val1DelAmt, true)
	tstaking.CreateValidator(val2Addr, valConsPk2, val1DelAmt, true)

	tstaking.Delegate(del1Addr, val1Addr, del1DelAmt)
	tstaking.Delegate(del2Addr, val1Addr, del2DelAmt)

	// assert correct initial balance
	expTokens := val1BalAmt.Sub(val1DelAmt)
	require.Equal(t,
		sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, expTokens)},
		app.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(val1Addr)),
	)

	// end block to bond validator
	staking.EndBlocker(ctx, app.StakingKeeper)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val1 := app.StakingKeeper.Validator(ctx, val1Addr)

	// allocate some rewards
	app.DistrKeeper.AllocateTokensToValidator(ctx, val1, sdk.DecCoins{sdk.NewDecCoin(sdk.DefaultBondDenom, allocatedReward)})

	require.Equal(t, uint64(6), app.DistrKeeper.GetValidatorHistoricalReferenceCount(ctx))

	// withdraw rewards

	// val1
	val1Reward, err := app.DistrKeeper.WithdrawDelegationRewards(ctx, sdk.AccAddress(val1Addr), val1Addr)
	require.NoError(t, err)
	require.Equal(t, val1FullReward.Mul(val1VestingRate).TruncateInt(), val1Reward.AmountOf(sdk.DefaultBondDenom))

	val1ExpBalance := val1BalAmt.Sub(val1DelAmt).Add(val1FullReward.Mul(val1VestingRate).TruncateInt())
	require.Equal(t,
		sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, val1ExpBalance)},
		app.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(val1Addr)),
	)

	require.Equal(t,
		sdk.DecCoins{sdk.NewDecCoinFromDec(sdk.DefaultBondDenom, val1FullReward.Sub(val1FullReward.Mul(val1VestingRate)))}, // part is locked
		app.DistrKeeper.GetDelegatorVestingLockedRewards(ctx, sdk.AccAddress(val1Addr)).Rewards,
	)

	// del1
	del1Reward, err := app.DistrKeeper.WithdrawDelegationRewards(ctx, del1Addr, val1Addr)
	require.NoError(t, err)
	require.Equal(t, del1FullReward.Mul(del1VestingRate).TruncateInt(), del1Reward.AmountOf(sdk.DefaultBondDenom))

	require.Equal(t,
		sdk.Coins{}, // zero, all coins are locked
		app.BankKeeper.GetAllBalances(ctx, del1Addr),
	)

	require.Equal(t,
		sdk.DecCoins{sdk.NewDecCoinFromDec(sdk.DefaultBondDenom, del1FullReward)}, // full reward is clocked
		app.DistrKeeper.GetDelegatorVestingLockedRewards(ctx, del1Addr).Rewards,
	)

	// del2
	del2Reward, err := app.DistrKeeper.WithdrawDelegationRewards(ctx, del2Addr, val1Addr)
	require.NoError(t, err)
	require.Equal(t, del2FullReward.Mul(del2VestingRate).TruncateInt(), del2Reward.AmountOf(sdk.DefaultBondDenom))

	del2ExpBalance := del2BalAmt.Sub(del2DelAmt).Add(del2FullReward.Mul(del2VestingRate).TruncateInt())
	require.Equal(t,
		sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, del2ExpBalance)}, // full reward
		app.BankKeeper.GetAllBalances(ctx, del2Addr),
	)

	require.Equal(t,
		sdk.DecCoins(nil), // nothing locked
		app.DistrKeeper.GetDelegatorVestingLockedRewards(ctx, del2Addr).Rewards,
	)

	// validator commission

	valCommissionReward, err := app.DistrKeeper.WithdrawValidatorCommission(ctx, val1Addr)
	require.Equal(t, allocatedReward.ToDec().Mul(val1Commission).TruncateInt(), valCommissionReward.AmountOf(sdk.DefaultBondDenom))
	require.NoError(t, err)

	// assert correct validator balance + commission
	valExpFinalBalance := val1BalAmt.Sub(val1DelAmt).
		Add(val1FullReward.Mul(val1VestingRate).TruncateInt()). // reward
		Add(allocatedReward.ToDec().Mul(val1Commission).TruncateInt()) // commission

	require.Equal(t,
		sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, valExpFinalBalance)},
		app.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(val1Addr)),
	)

	// ----- withdraw the locked reward

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Minute)) // increase time to pass the vesting periods

	// fetch validator and delegation one more time
	val1 = app.StakingKeeper.Validator(ctx, val1Addr)

	require.Equal(t, uint64(6), app.DistrKeeper.GetValidatorHistoricalReferenceCount(ctx))

	// val1
	val1Reward, err = app.DistrKeeper.WithdrawDelegationRewards(ctx, sdk.AccAddress(val1Addr), val1Addr)
	require.NoError(t, err)
	require.Equal(t, val1FullReward.Sub(val1FullReward.Mul(val1VestingRate)).TruncateInt(), val1Reward.AmountOf(sdk.DefaultBondDenom))

	val1ExpBalance = val1BalAmt.Sub(val1DelAmt).
		Add(val1FullReward.TruncateInt()). // full reward
		Add(allocatedReward.ToDec().Mul(val1Commission).TruncateInt()) // commission
	require.Equal(t,
		sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, val1ExpBalance)},
		app.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(val1Addr)),
	)

	require.Equal(t,
		sdk.DecCoins(nil), // all locked cons are withdrawn
		app.DistrKeeper.GetDelegatorVestingLockedRewards(ctx, sdk.AccAddress(val1Addr)).Rewards,
	)

	// del1
	del1Reward, err = app.DistrKeeper.WithdrawDelegationRewards(ctx, del1Addr, val1Addr)
	require.NoError(t, err)
	require.Equal(t, del1FullReward.TruncateInt(), del1Reward.AmountOf(sdk.DefaultBondDenom)) // full reward will come from the locked reward

	require.Equal(t,
		sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, del1FullReward.TruncateInt())),
		app.BankKeeper.GetAllBalances(ctx, del1Addr),
	)

	require.Equal(t,
		sdk.DecCoins(nil), // all locked cons are withdrawn
		app.DistrKeeper.GetDelegatorVestingLockedRewards(ctx, del1Addr).Rewards,
	)

	// del2
	del2Reward, err = app.DistrKeeper.WithdrawDelegationRewards(ctx, del2Addr, val1Addr)
	require.NoError(t, err)
	require.Equal(t, sdk.ZeroInt(), del2Reward.AmountOf(sdk.DefaultBondDenom))

	del2ExpBalance = del2BalAmt.Sub(del2DelAmt).Add(del2FullReward.Mul(del2VestingRate).TruncateInt())
	require.Equal(t,
		sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, del2ExpBalance)}, // full reward
		app.BankKeeper.GetAllBalances(ctx, del2Addr),
	)

	require.Equal(t,
		sdk.DecCoins(nil), // all locked cons are withdrawn
		app.DistrKeeper.GetDelegatorVestingLockedRewards(ctx, del2Addr).Rewards,
	)
}

func TestWithdrawDelegationRewardsVestingPeriods(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	valBalAmt := app.StakingKeeper.TokensFromConsensusPower(ctx, 1000)
	valVestingPeriodAmt := valBalAmt.QuoRaw(4)
	valDelAmt := app.StakingKeeper.TokensFromConsensusPower(ctx, 100)
	valCommission := sdk.ZeroDec()

	allocatedPerStepReward := app.StakingKeeper.TokensFromConsensusPower(ctx, 200)
	allocatedFullReward := allocatedPerStepReward.MulRaw(2)

	valAddr := simapp.ConvertAddrsToValAddrs(simapp.AddTestVestingAddrs(app, ctx, 1, valBalAmt,
		valVestingPeriodAmt, valVestingPeriodAmt, valVestingPeriodAmt))[0] // 25% unlocked at the beginning and 3 more periods with 25% each

	tstaking := teststaking.NewHelper(t, ctx, app.StakingKeeper)

	// set module account coins
	distrAcc := app.DistrKeeper.GetDistributionAccount(ctx)
	require.NoError(t, simapp.FundModuleAccount(app.BankKeeper, ctx, distrAcc.GetName(), sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, allocatedFullReward))))
	app.AccountKeeper.SetModuleAccount(ctx, distrAcc)

	// create validator with 0% commission
	tstaking.Commission = stakingtypes.NewCommissionRates(valCommission, valCommission, sdk.NewDec(0))
	tstaking.CreateValidator(valAddr, valConsPk1, valDelAmt, true)

	// assert correct initial balance
	expTokens := valBalAmt.Sub(valDelAmt)
	require.Equal(t,
		sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, expTokens)},
		app.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(valAddr)),
	)

	// end block to bond validator
	staking.EndBlocker(ctx, app.StakingKeeper)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// rewards

	// allocate and withdraw the reward first time

	// allocate some rewards
	val := app.StakingKeeper.Validator(ctx, valAddr)
	app.DistrKeeper.AllocateTokensToValidator(ctx, val, sdk.DecCoins{sdk.NewDecCoin(sdk.DefaultBondDenom, allocatedPerStepReward)})

	valReward, err := app.DistrKeeper.WithdrawDelegationRewards(ctx, sdk.AccAddress(valAddr), valAddr)
	expReward := allocatedPerStepReward.ToDec().Mul(valVestingPeriodAmt.ToDec().QuoInt(valBalAmt)).TruncateInt()
	require.NoError(t, err)
	require.Equal(t, expReward, valReward.AmountOf(sdk.DefaultBondDenom))

	lockedStep1 := allocatedPerStepReward.Sub(expReward)
	require.Equal(t,
		sdk.NewDecCoins(sdk.NewDecCoin(sdk.DefaultBondDenom, lockedStep1)),
		app.DistrKeeper.GetDelegatorVestingLockedRewards(ctx, sdk.AccAddress(valAddr)).Rewards,
	)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// no reward since the vested ratio is the same
	valReward, err = app.DistrKeeper.WithdrawDelegationRewards(ctx, sdk.AccAddress(valAddr), valAddr)
	require.NoError(t, err)
	require.Equal(t, sdk.ZeroInt(), valReward.AmountOf(sdk.DefaultBondDenom))

	lockedStep2 := allocatedPerStepReward.Sub(expReward) // the locked amount should be the same
	require.Equal(t,
		sdk.NewDecCoins(sdk.NewDecCoin(sdk.DefaultBondDenom, lockedStep2)),
		app.DistrKeeper.GetDelegatorVestingLockedRewards(ctx, sdk.AccAddress(valAddr)).Rewards,
	)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(2 * time.Second)) // increase to reach next period

	valReward, err = app.DistrKeeper.WithdrawDelegationRewards(ctx, sdk.AccAddress(valAddr), valAddr)
	require.NoError(t, err)
	// the reward is quarter of the locked reward
	require.Equal(t, lockedStep2.QuoRaw(4), valReward.AmountOf(sdk.DefaultBondDenom))

	lockedStep3 := lockedStep2.Sub(lockedStep2.QuoRaw(4)) // the locked reward is decreased should be the same
	require.Equal(t,
		sdk.NewDecCoins(sdk.NewDecCoin(sdk.DefaultBondDenom, lockedStep3)),
		app.DistrKeeper.GetDelegatorVestingLockedRewards(ctx, sdk.AccAddress(valAddr)).Rewards,
	)

	// allocate more rewards
	val = app.StakingKeeper.Validator(ctx, valAddr)
	app.DistrKeeper.AllocateTokensToValidator(ctx, val, sdk.DecCoins{sdk.NewDecCoin(sdk.DefaultBondDenom, allocatedPerStepReward)})

	// next block and increase the vested
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(3 * time.Second)) // increase to reach next period

	valReward, err = app.DistrKeeper.WithdrawDelegationRewards(ctx, sdk.AccAddress(valAddr), valAddr)
	require.NoError(t, err)
	// allocatedPerStepReward * 3/4 + (1/4 of the prev locked reward)
	require.Equal(t, allocatedPerStepReward.ToDec().MulInt64(3).QuoInt64(4).Add(lockedStep3.ToDec().QuoInt64(4)).TruncateInt(), valReward.AmountOf(sdk.DefaultBondDenom))

	// - 1/4 of the prev locked and add 1/4 of the allocatedPerStepReward
	lockedStep4 := lockedStep3.Sub(lockedStep3.QuoRaw(4)).Add(allocatedPerStepReward.QuoRaw(4))
	require.Equal(t,
		sdk.NewDecCoins(sdk.NewDecCoin(sdk.DefaultBondDenom, lockedStep4)),
		app.DistrKeeper.GetDelegatorVestingLockedRewards(ctx, sdk.AccAddress(valAddr)).Rewards,
	)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Minute)) // unlock all vested

	valReward, err = app.DistrKeeper.WithdrawDelegationRewards(ctx, sdk.AccAddress(valAddr), valAddr)
	require.NoError(t, err)
	// the reward is the prev locked amount
	require.Equal(t, lockedStep4, valReward.AmountOf(sdk.DefaultBondDenom))

	require.Equal(t,
		sdk.DecCoins(nil), // all coins are unlocked
		app.DistrKeeper.GetDelegatorVestingLockedRewards(ctx, sdk.AccAddress(valAddr)).Rewards,
	)

	require.Equal(t,
		sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, valBalAmt.Add(allocatedFullReward).Sub(valDelAmt))}, // full reward received
		app.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(valAddr)),
	)

}

func TestWithdrawDelegationRewardsVestingMultipleValidators(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	valCommission := sdk.NewDecWithPrec(5, 1) // 50%

	val1BalAmt := app.StakingKeeper.TokensFromConsensusPower(ctx, 1000)
	val1DelAmt := app.StakingKeeper.TokensFromConsensusPower(ctx, 100)

	val2BalAmt := app.StakingKeeper.TokensFromConsensusPower(ctx, 1000)
	val2DelAmt := app.StakingKeeper.TokensFromConsensusPower(ctx, 100)

	delBalAmt := app.StakingKeeper.TokensFromConsensusPower(ctx, 1000)
	delVestingAmt := delBalAmt.QuoRaw(2) // 50% in vesting
	delVal1DelAmt := delBalAmt.QuoRaw(4) // 25% to val1
	delVal2DelAmt := delBalAmt.QuoRaw(4) // 25% to val2
	delUnbondedBalAmt := delBalAmt.Sub(delVal1DelAmt).Sub(delVal2DelAmt)
	delVestingRate := sdk.OneDec().Sub(delVestingAmt.ToDec().Quo(delBalAmt.ToDec())) // 50% locked in vesting

	val1AllocatedReward := app.StakingKeeper.TokensFromConsensusPower(ctx, 250)
	val2AllocatedReward := app.StakingKeeper.TokensFromConsensusPower(ctx, 250)
	allocatedReward := val1AllocatedReward.Add(val2AllocatedReward)

	delVal1FullReward := delVal1DelAmt.ToDec().Quo(delVal1DelAmt.Add(val1DelAmt).ToDec()).Mul(val1AllocatedReward.ToDec().Mul(valCommission))
	delVal2FullReward := delVal2DelAmt.ToDec().Quo(delVal2DelAmt.Add(val2DelAmt).ToDec()).Mul(val2AllocatedReward.ToDec().Mul(valCommission))

	val1Addr := simapp.ConvertAddrsToValAddrs(simapp.AddTestAddrs(app, ctx, 1, val1BalAmt))[0]
	val2Addr := simapp.ConvertAddrsToValAddrs(simapp.AddTestAddrs(app, ctx, 1, val2BalAmt))[0]

	delAddr := simapp.AddTestVestingAddrs(app, ctx, 1, delBalAmt, delVestingAmt)[0]

	tstaking := teststaking.NewHelper(t, ctx, app.StakingKeeper)

	// set module account coins
	distrAcc := app.DistrKeeper.GetDistributionAccount(ctx)
	require.NoError(t, simapp.FundModuleAccount(app.BankKeeper, ctx, distrAcc.GetName(), sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, allocatedReward))))
	app.AccountKeeper.SetModuleAccount(ctx, distrAcc)

	// create validator with 50% commission
	tstaking.Commission = stakingtypes.NewCommissionRates(valCommission, valCommission, sdk.NewDec(0))
	tstaking.CreateValidator(val1Addr, valConsPk1, val1DelAmt, true)
	tstaking.CreateValidator(val2Addr, valConsPk2, val2DelAmt, true)

	tstaking.Delegate(delAddr, val1Addr, delVal1DelAmt)
	tstaking.Delegate(delAddr, val2Addr, delVal2DelAmt)

	// end block to bond validator
	staking.EndBlocker(ctx, app.StakingKeeper)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val1 := app.StakingKeeper.Validator(ctx, val1Addr)
	val2 := app.StakingKeeper.Validator(ctx, val2Addr)

	// allocate some rewards
	app.DistrKeeper.AllocateTokensToValidator(ctx, val1, sdk.DecCoins{sdk.NewDecCoin(sdk.DefaultBondDenom, val1AllocatedReward)})
	app.DistrKeeper.AllocateTokensToValidator(ctx, val2, sdk.DecCoins{sdk.NewDecCoin(sdk.DefaultBondDenom, val2AllocatedReward)})

	require.Equal(t, uint64(6), app.DistrKeeper.GetValidatorHistoricalReferenceCount(ctx))

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// withdraw rewards

	// del val1
	delVal1Reward, err := app.DistrKeeper.WithdrawDelegationRewards(ctx, delAddr, val1Addr)
	require.NoError(t, err)
	require.Equal(t, delVal1FullReward.Mul(delVestingRate).TruncateInt(), delVal1Reward.AmountOf(sdk.DefaultBondDenom))

	delVal1LockedReward := app.DistrKeeper.GetDelegatorVestingLockedRewards(ctx, delAddr).Rewards.AmountOf(sdk.DefaultBondDenom)
	require.Equal(t, delVal1FullReward.Mul(delVestingRate).TruncateInt(), delVal1LockedReward.TruncateInt())

	// del val2
	delVal2Reward, err := app.DistrKeeper.WithdrawDelegationRewards(ctx, delAddr, val2Addr)
	require.NoError(t, err)
	require.Equal(t, delVal2FullReward.Mul(delVestingRate).TruncateInt(), delVal2Reward.AmountOf(sdk.DefaultBondDenom))

	delVal2LockedReward := app.DistrKeeper.GetDelegatorVestingLockedRewards(ctx, delAddr).Rewards.AmountOf(sdk.DefaultBondDenom)
	// the locked reward is doubled now
	require.Equal(t, delVal1FullReward.Mul(delVestingRate).Add(delVal2FullReward.Mul(delVestingRate)).TruncateInt(), delVal2LockedReward.TruncateInt())

	// ----- withdraw the locked reward

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Minute)) // increase time to pass the vesting periods

	_, err = app.DistrKeeper.WithdrawDelegationRewards(ctx, delAddr, val1Addr)
	require.NoError(t, err)

	// no locked coins must be here now
	require.Equal(t,
		sdk.DecCoins(nil), // all locked cons are withdrawn
		app.DistrKeeper.GetDelegatorVestingLockedRewards(ctx, delAddr).Rewards,
	)

	// the final balance is initial + withdrawn reward
	require.Equal(t,
		sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, delUnbondedBalAmt.Add(delVal1FullReward.TruncateInt()).Add(delVal2FullReward.TruncateInt()))},
		app.BankKeeper.GetAllBalances(ctx, delAddr),
	)
}

func TestCalculateRewardsAfterManySlashesInSameBlock(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	addr := simapp.AddTestAddrs(app, ctx, 1, sdk.NewInt(1000000000))
	valAddrs := simapp.ConvertAddrsToValAddrs(addr)
	tstaking := teststaking.NewHelper(t, ctx, app.StakingKeeper)

	// create validator with 50% commission
	valPower := int64(100)
	tstaking.Commission = stakingtypes.NewCommissionRates(sdk.NewDecWithPrec(5, 1), sdk.NewDecWithPrec(5, 1), sdk.NewDec(0))
	tstaking.CreateValidatorWithValPower(valAddrs[0], valConsPk1, valPower, true)

	// end block to bond validator
	staking.EndBlocker(ctx, app.StakingKeeper)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val := app.StakingKeeper.Validator(ctx, valAddrs[0])
	del := app.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddrs[0]), valAddrs[0])

	// end period
	endingPeriod := app.DistrKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards
	rewards := app.DistrKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)

	// rewards should be zero
	require.True(t, rewards.IsZero())

	// start out block height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// allocate some rewards
	initial := app.StakingKeeper.TokensFromConsensusPower(ctx, 10).ToDec()
	tokens := sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: initial}}
	app.DistrKeeper.AllocateTokensToValidator(ctx, val, tokens)

	// slash the validator by 50%
	app.StakingKeeper.Slash(ctx, valConsAddr1, ctx.BlockHeight(), valPower, sdk.NewDecWithPrec(5, 1))

	// slash the validator by 50% again
	app.StakingKeeper.Slash(ctx, valConsAddr1, ctx.BlockHeight(), valPower/2, sdk.NewDecWithPrec(5, 1))

	// fetch the validator again
	val = app.StakingKeeper.Validator(ctx, valAddrs[0])

	// increase block height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// allocate some more rewards
	app.DistrKeeper.AllocateTokensToValidator(ctx, val, tokens)

	// end period
	endingPeriod = app.DistrKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards
	rewards = app.DistrKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)

	// rewards should be half the tokens
	require.Equal(t, sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: initial}}, rewards)

	// commission should be the other half
	require.Equal(t, sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: initial}}, app.DistrKeeper.GetValidatorAccumulatedCommission(ctx, valAddrs[0]).Commission)
}

func TestCalculateRewardsMultiDelegatorMultiSlash(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	tstaking := teststaking.NewHelper(t, ctx, app.StakingKeeper)
	addr := simapp.AddTestAddrs(app, ctx, 2, sdk.NewInt(1000000000))
	valAddrs := simapp.ConvertAddrsToValAddrs(addr)

	// create validator with 50% commission
	tstaking.Commission = stakingtypes.NewCommissionRates(sdk.NewDecWithPrec(5, 1), sdk.NewDecWithPrec(5, 1), sdk.NewDec(0))
	valPower := int64(100)
	tstaking.CreateValidatorWithValPower(valAddrs[0], valConsPk1, valPower, true)

	// end block to bond validator
	staking.EndBlocker(ctx, app.StakingKeeper)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val := app.StakingKeeper.Validator(ctx, valAddrs[0])
	del1 := app.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddrs[0]), valAddrs[0])

	// allocate some rewards
	initial := app.StakingKeeper.TokensFromConsensusPower(ctx, 30).ToDec()
	tokens := sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: initial}}
	app.DistrKeeper.AllocateTokensToValidator(ctx, val, tokens)

	// slash the validator
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)
	app.StakingKeeper.Slash(ctx, valConsAddr1, ctx.BlockHeight(), valPower, sdk.NewDecWithPrec(5, 1))
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// second delegation
	tstaking.DelegateWithPower(sdk.AccAddress(valAddrs[1]), valAddrs[0], 100)

	del2 := app.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddrs[1]), valAddrs[0])

	// end block
	staking.EndBlocker(ctx, app.StakingKeeper)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// allocate some more rewards
	app.DistrKeeper.AllocateTokensToValidator(ctx, val, tokens)

	// slash the validator again
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)
	app.StakingKeeper.Slash(ctx, valConsAddr1, ctx.BlockHeight(), valPower, sdk.NewDecWithPrec(5, 1))
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// fetch updated validator
	val = app.StakingKeeper.Validator(ctx, valAddrs[0])

	// end period
	endingPeriod := app.DistrKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards for del1
	rewards := app.DistrKeeper.CalculateDelegationRewards(ctx, val, del1, endingPeriod)

	// rewards for del1 should be 2/3 initial (half initial first period, 1/6 initial second period)
	require.Equal(t, sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: initial.QuoInt64(2).Add(initial.QuoInt64(6))}}, rewards)

	// calculate delegation rewards for del2
	rewards = app.DistrKeeper.CalculateDelegationRewards(ctx, val, del2, endingPeriod)

	// rewards for del2 should be initial / 3
	require.Equal(t, sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: initial.QuoInt64(3)}}, rewards)

	// commission should be equal to initial (twice 50% commission, unaffected by slashing)
	require.Equal(t, sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: initial}}, app.DistrKeeper.GetValidatorAccumulatedCommission(ctx, valAddrs[0]).Commission)
}

func TestCalculateRewardsMultiDelegatorMultWithdraw(t *testing.T) {
	app := simapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	tstaking := teststaking.NewHelper(t, ctx, app.StakingKeeper)
	addr := simapp.AddTestAddrs(app, ctx, 2, sdk.NewInt(1000000000))
	valAddrs := simapp.ConvertAddrsToValAddrs(addr)
	initial := int64(20)

	// set module account coins
	distrAcc := app.DistrKeeper.GetDistributionAccount(ctx)
	require.NoError(t, simapp.FundModuleAccount(app.BankKeeper, ctx, distrAcc.GetName(), sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(1000)))))
	app.AccountKeeper.SetModuleAccount(ctx, distrAcc)

	tokens := sdk.DecCoins{sdk.NewDecCoinFromDec(sdk.DefaultBondDenom, sdk.NewDec(initial))}

	// create validator with 50% commission
	tstaking.Commission = stakingtypes.NewCommissionRates(sdk.NewDecWithPrec(5, 1), sdk.NewDecWithPrec(5, 1), sdk.NewDec(0))
	tstaking.CreateValidator(valAddrs[0], valConsPk1, sdk.NewInt(100), true)

	// end block to bond validator
	staking.EndBlocker(ctx, app.StakingKeeper)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val := app.StakingKeeper.Validator(ctx, valAddrs[0])
	del1 := app.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddrs[0]), valAddrs[0])

	// allocate some rewards
	app.DistrKeeper.AllocateTokensToValidator(ctx, val, tokens)

	// historical count should be 2 (validator init, delegation init)
	require.Equal(t, uint64(2), app.DistrKeeper.GetValidatorHistoricalReferenceCount(ctx))

	// second delegation
	tstaking.Delegate(sdk.AccAddress(valAddrs[1]), valAddrs[0], sdk.NewInt(100))

	// historical count should be 3 (second delegation init)
	require.Equal(t, uint64(3), app.DistrKeeper.GetValidatorHistoricalReferenceCount(ctx))

	// fetch updated validator
	val = app.StakingKeeper.Validator(ctx, valAddrs[0])
	del2 := app.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddrs[1]), valAddrs[0])

	// end block
	staking.EndBlocker(ctx, app.StakingKeeper)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// allocate some more rewards
	app.DistrKeeper.AllocateTokensToValidator(ctx, val, tokens)

	// first delegator withdraws
	_, err := app.DistrKeeper.WithdrawDelegationRewards(ctx, sdk.AccAddress(valAddrs[0]), valAddrs[0])
	require.NoError(t, err)

	// second delegator withdraws
	_, err = app.DistrKeeper.WithdrawDelegationRewards(ctx, sdk.AccAddress(valAddrs[1]), valAddrs[0])
	require.NoError(t, err)

	// historical count should be 3 (validator init + two delegations)
	require.Equal(t, uint64(3), app.DistrKeeper.GetValidatorHistoricalReferenceCount(ctx))

	// validator withdraws commission
	_, err = app.DistrKeeper.WithdrawValidatorCommission(ctx, valAddrs[0])
	require.NoError(t, err)

	// end period
	endingPeriod := app.DistrKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards for del1
	rewards := app.DistrKeeper.CalculateDelegationRewards(ctx, val, del1, endingPeriod)

	// rewards for del1 should be zero
	require.True(t, rewards.IsZero())

	// calculate delegation rewards for del2
	rewards = app.DistrKeeper.CalculateDelegationRewards(ctx, val, del2, endingPeriod)

	// rewards for del2 should be zero
	require.True(t, rewards.IsZero())

	// commission should be zero
	require.True(t, app.DistrKeeper.GetValidatorAccumulatedCommission(ctx, valAddrs[0]).Commission.IsZero())

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// allocate some more rewards
	app.DistrKeeper.AllocateTokensToValidator(ctx, val, tokens)

	// first delegator withdraws again
	_, err = app.DistrKeeper.WithdrawDelegationRewards(ctx, sdk.AccAddress(valAddrs[0]), valAddrs[0])
	require.NoError(t, err)

	// end period
	endingPeriod = app.DistrKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards for del1
	rewards = app.DistrKeeper.CalculateDelegationRewards(ctx, val, del1, endingPeriod)

	// rewards for del1 should be zero
	require.True(t, rewards.IsZero())

	// calculate delegation rewards for del2
	rewards = app.DistrKeeper.CalculateDelegationRewards(ctx, val, del2, endingPeriod)

	// rewards for del2 should be 1/4 initial
	require.Equal(t, sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: sdk.NewDec(initial / 4)}}, rewards)

	// commission should be half initial
	require.Equal(t, sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: sdk.NewDec(initial / 2)}}, app.DistrKeeper.GetValidatorAccumulatedCommission(ctx, valAddrs[0]).Commission)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// allocate some more rewards
	app.DistrKeeper.AllocateTokensToValidator(ctx, val, tokens)

	// withdraw commission
	_, err = app.DistrKeeper.WithdrawValidatorCommission(ctx, valAddrs[0])
	require.NoError(t, err)

	// end period
	endingPeriod = app.DistrKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards for del1
	rewards = app.DistrKeeper.CalculateDelegationRewards(ctx, val, del1, endingPeriod)

	// rewards for del1 should be 1/4 initial
	require.Equal(t, sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: sdk.NewDec(initial / 4)}}, rewards)

	// calculate delegation rewards for del2
	rewards = app.DistrKeeper.CalculateDelegationRewards(ctx, val, del2, endingPeriod)

	// rewards for del2 should be 1/2 initial
	require.Equal(t, sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: sdk.NewDec(initial / 2)}}, rewards)

	// commission should be zero
	require.True(t, app.DistrKeeper.GetValidatorAccumulatedCommission(ctx, valAddrs[0]).Commission.IsZero())
}
