<!--
order: 3
-->

# Begin-Block

Minting parameters are recalculated and inflation
paid at the beginning of each block.

## NextInflationRate

The target annual inflation rate is recalculated each block.
Initially, the annual inflation rate follows a hyper-inflationary curve
Once, the hyper-inflationary period is finished then inflation will
adjust according to percentage staked.

During the stabilization period, inflation is subject to a rate change 
(positive or negative) depending on the distance from the desired ratio (67%).
The maximum rate change possible is defined to be 13% per year, however the 
annual inflation is capped as between 7% and 20%.

```
func (m Minter) NextInflationRate(params Params, bondedRatio sdk.Dec, totalSupply sdk.Int) sdk.Dec {
	// NOM staking is defined by an initial hyper-inflationary regime followed by an
	// infinite regime stabilizing % staked around a goal.

	// End of the hyper-inflationary period
	endHyperInflation := sdk.NewInt(int64(25000000))

	// Initialize the inflation variable
	inflation := sdk.NewDec(int64(0))

	if totalSupply.GTE(endHyperInflation) {
		// Infinite stabilized regime
		//
		// The target annual inflation rate is recalculated for each previsions cycle. The
		// inflation is also subject to a rate change (positive or negative) depending on
		// the distance from the desired ratio (67%). The maximum rate change possible is
		// defined to be 13% per year, however the annual inflation is capped as between
		// 7% and 20%.

		// (1 - bondedRatio/GoalBonded) * InflationRateChange
		inflationRateChangePerYear := sdk.OneDec().
			Sub(bondedRatio.Quo(params.GoalBonded)).
			Mul(params.InflationRateChange)
		inflationRateChange := inflationRateChangePerYear.Quo(sdk.NewDec(int64(params.BlocksPerYear)))

		// adjust the new annual inflation for this next cycle
		inflation = m.Inflation.Add(inflationRateChange) // note inflationRateChange may be negative
		if inflation.GT(params.InflationMax) {
			inflation = params.InflationMax
		}
		if inflation.LT(params.InflationMin) {
			inflation = params.InflationMin
		}

	} else {
		// Hyper-inflationary regime

		totalSupplyDec := totalSupply.ToDec()

		// Using the Irwin-Hall Distribution
		// First Leg: 0 < supply < 22000000
		// function: a(b*abs((supply-e)/f)^3 - c*((supply-e)/f)^2 + d)
		//
		// Second Leg: 22000000 < supply < 25000000
		// function: a(g - (supply-e)/f)^3

		a := sdk.NewDecWithPrec(int64(25), int64(2))
		// Position of Peak
		e := sdk.NewDec(int64(12000000))
		// Width of Bell
		f := sdk.NewDec(int64(10000000))

		// fmt.Println("TotalSupply: ", totalSupplyDec)

		if totalSupply.LTE(sdk.NewInt(int64(22000000))) {
			// First Leg
			b := sdk.NewDec(int64(3))
			c := sdk.NewDec(int64(6))
			d := sdk.NewDec(int64(4))

			// abs((supply-e)/f)
			temp1 := totalSupplyDec.Sub(e).Quo(f).Abs()

			// b*abs((supply-e)/f)^3
			temp2 := b.Mul(temp1.Power(uint64(3)))

			// c*((supply-e)/f)^2
			temp3 := c.Mul(totalSupplyDec.Sub(e).Quo(f).Power(uint64(2)))

			// function: a(b*abs((supply-e)/f)^3 - c*((supply-e)/f)^2 + d)
			inflation = a.Mul(temp2.Sub(temp3).Add(d))

			fmt.Println("Inflation: ", inflation)
		} else {
			// Second Leg
			g := sdk.NewDec(int64(2))

			// function: a(g - (supply-e)/f)^3
			inflation = a.Mul(g.Sub(totalSupplyDec.Sub(e).Quo(f)).Power(uint64(3)))
		}

	}

	return inflation
}
```

## NextAnnualProvisions

Calculate the annual provisions based on current total supply and inflation
rate. This parameter is calculated once per block. 

```
NextAnnualProvisions(params Params, totalSupply sdk.Dec) (provisions sdk.Dec) {
	return Inflation * totalSupply
```

## BlockProvision

Calculate the provisions generated for each block based on current annual provisions. The provisions are then minted by the `mint` module's `ModuleMinterAccount` and then transferred to the `auth`'s `FeeCollector` `ModuleAccount`.

```
BlockProvision(params Params) sdk.Coin {
	provisionAmt = AnnualProvisions/ params.BlocksPerYear
	return sdk.NewCoin(params.MintDenom, provisionAmt.Truncate())
```
