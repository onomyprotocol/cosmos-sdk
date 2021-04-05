package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewMinter returns a new Minter object with the given inflation and annual
// provisions values.
func NewMinter(inflation, annualProvisions sdk.Dec) Minter {
	return Minter{
		Inflation:        inflation,
		AnnualProvisions: annualProvisions,
	}
}

// InitialMinter returns an initial Minter object with a given inflation value.
func InitialMinter(inflation sdk.Dec) Minter {
	return NewMinter(
		inflation,
		sdk.NewDec(0),
	)
}

// DefaultInitialMinter returns a default initial Minter object for a new chain
// which uses an inflation rate of 13%.
func DefaultInitialMinter() Minter {
	return InitialMinter(
		sdk.NewDecWithPrec(13, 2),
	)
}

// validate minter
func ValidateMinter(minter Minter) error {
	if minter.Inflation.IsNegative() {
		return fmt.Errorf("mint parameter Inflation should be positive, is %s",
			minter.Inflation.String())
	}
	return nil
}

// NextInflationRate returns the new inflation rate for the next hour.
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

// NextAnnualProvisions returns the annual provisions based on current total
// supply and inflation rate.
func (m Minter) NextAnnualProvisions(_ Params, totalSupply sdk.Int) sdk.Dec {
	return m.Inflation.MulInt(totalSupply)
}

// BlockProvision returns the provisions for a block based on the annual
// provisions rate.
func (m Minter) BlockProvision(params Params) sdk.Coin {
	provisionAmt := m.AnnualProvisions.QuoInt(sdk.NewInt(int64(params.BlocksPerYear)))
	return sdk.NewCoin(params.MintDenom, provisionAmt.TruncateInt())
}
