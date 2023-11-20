package keeper

import (
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// AllocateTokens performs reward and fee distribution to all validators based
// on the F1 fee distribution specification.
func (k Keeper) AllocateTokens(ctx sdk.Context, totalPreviousPower int64, bondedVotes []abci.VoteInfo) {
	// fetch and clear the collected fees for distribution, since this is
	// called in BeginBlock, collected fees will be from the previous block
	// (and distributed to the previous proposer)
	feeCollector := k.authKeeper.GetModuleAccount(ctx, k.feeCollectorName)
	feesCollectedInt := k.bankKeeper.GetAllBalances(ctx, feeCollector.GetAddress())
	feesCollected := sdk.NewDecCoinsFromCoins(feesCollectedInt...)

	// transfer collected fees to the distribution module account
	err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, k.feeCollectorName, types.ModuleName, feesCollectedInt)
	if err != nil {
		panic(err)
	}

	// temporary workaround to keep CanWithdrawInvariant happy
	// general discussions here: https://github.com/cosmos/cosmos-sdk/issues/2906#issuecomment-441867634
	feePool := k.GetFeePool(ctx)
	if totalPreviousPower == 0 {
		feePool.CommunityPool = feePool.CommunityPool.Add(feesCollected...)
		k.SetFeePool(ctx, feePool)
		return
	}

	// calculate fraction allocated to validators
	remaining := feesCollected
	communityTax := k.GetCommunityTax(ctx)
	voteMultiplier := math.LegacyOneDec().Sub(communityTax)
	feeMultiplier := feesCollected.MulDecTruncate(voteMultiplier)

	// allocate tokens proportionally to voting power
	//
	// TODO: Consider parallelizing later
	//
	// Ref: https://github.com/cosmos/cosmos-sdk/pull/3099#discussion_r246276376
	for _, vote := range bondedVotes {
		validator := k.stakingKeeper.ValidatorByConsAddr(ctx, vote.Validator.Address)
		// TODO: Consider micro-slashing for missing votes.
		//
		// Ref: https://github.com/cosmos/cosmos-sdk/issues/2525#issuecomment-430838701
		powerFraction := math.LegacyNewDec(vote.Validator.Power).QuoTruncate(math.LegacyNewDec(totalPreviousPower))
		reward := feeMultiplier.MulDecTruncate(powerFraction)

		k.AllocateTokensToValidator(ctx, validator, reward)
		remaining = remaining.Sub(reward)
	}

	// allocate community funding
	feePool.CommunityPool = feePool.CommunityPool.Add(remaining...)
	k.SetFeePool(ctx, feePool)
}

// AllocateTokensToValidator allocate tokens to a particular validator,
// splitting according to commission.
func (k Keeper) AllocateTokensToValidator(ctx sdk.Context, val stakingtypes.ValidatorI, tokens sdk.DecCoins) {
	// split tokens between validator and delegators according to commission
	//commission := tokens.MulDec(val.GetCommission())
	//shared := tokens.Sub(commission)

	//// update current commission
	//ctx.EventManager().EmitEvent(
	//	sdk.NewEvent(
	//		types.EventTypeCommission,
	//		sdk.NewAttribute(sdk.AttributeKeyAmount, commission.String()),
	//		sdk.NewAttribute(types.AttributeKeyValidator, val.GetOperator().String()),
	//	),
	//)
	//currentCommission := k.GetValidatorAccumulatedCommission(ctx, val.GetOperator())
	//currentCommission.Commission = currentCommission.Commission.Add(commission...)
	//k.SetValidatorAccumulatedCommission(ctx, val.GetOperator(), currentCommission)

	logger := ctx.Logger()

	var valBurn = val
	params := k.GetParams(ctx)
	logger.Debug("[Distribution] allocate tokens", "Validator", val.GetOperator().String(), "reward", tokens.String(), "params", params)
	if params.BurnAddress != "" {
		for _, v := range params.BurnValidators {
			if v == val.GetOperator().String() {
				burnVal, err := sdk.ValAddressFromBech32(params.BurnAddress)
				if err != nil {
					panic("burn address is not a valid operator address")
				}
				valBurn = k.stakingKeeper.Validator(ctx, burnVal)
				logger.Info("[Distribution] burn tokens", "Validator", val.GetOperator().String(), "reward", tokens.String(), "burn address", valBurn.GetOperator())
			}
		}
	}

	// update current rewards
	currentRewards := k.GetValidatorCurrentRewards(ctx, valBurn.GetOperator())
	currentRewards.Rewards = currentRewards.Rewards.Add(tokens...)
	k.SetValidatorCurrentRewards(ctx, valBurn.GetOperator(), currentRewards)

	// update outstanding rewards
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRewards,
			sdk.NewAttribute(sdk.AttributeKeyAmount, tokens.String()),
			sdk.NewAttribute(types.AttributeKeyValidator, valBurn.GetOperator().String()),
		),
	)

	outstanding := k.GetValidatorOutstandingRewards(ctx, valBurn.GetOperator())
	outstanding.Rewards = outstanding.Rewards.Add(tokens...)
	k.SetValidatorOutstandingRewards(ctx, valBurn.GetOperator(), outstanding)
}
