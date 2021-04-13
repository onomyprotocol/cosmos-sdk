package crisis

import (
	"time"

	"github.com/onomyprotocol/cosmos-sdk/telemetry"
	sdk "github.com/onomyprotocol/cosmos-sdk/types"
	"github.com/onomyprotocol/cosmos-sdk/x/crisis/keeper"
	"github.com/onomyprotocol/cosmos-sdk/x/crisis/types"
)

// check all registered invariants
func EndBlocker(ctx sdk.Context, k keeper.Keeper) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	if k.InvCheckPeriod() == 0 || ctx.BlockHeight()%int64(k.InvCheckPeriod()) != 0 {
		// skip running the invariant check
		return
	}
	k.AssertInvariants(ctx)
}
