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
	params := k.GetParams(ctx)
	for _, vote := range bondedVotes {
		validator := k.stakingKeeper.ValidatorByConsAddr(ctx, vote.Validator.Address)
		// TODO: Consider micro-slashing for missing votes.
		//
		// Ref: https://github.com/cosmos/cosmos-sdk/issues/2525#issuecomment-430838701
		powerFraction := math.LegacyNewDec(vote.Validator.Power).QuoTruncate(math.LegacyNewDec(totalPreviousPower))
		reward := feeMultiplier.MulDecTruncate(powerFraction)
		k.burnOrAllocateTokensToValidator(ctx, validator, params, reward)
		remaining = remaining.Sub(reward)
	}

	// allocate community funding
	feePool.CommunityPool = feePool.CommunityPool.Add(remaining...)
	k.SetFeePool(ctx, feePool)
}

func (k Keeper) burnOrAllocateTokensToValidator(ctx sdk.Context, validator stakingtypes.ValidatorI, params types.Params, reward sdk.DecCoins) {
	var err error
	logger := ctx.Logger()
	for _, v := range params.BurnValidators {
		if v == validator.GetOperator().String() {
			var coins sdk.Coins
			for _, r := range reward {
				coins = append(coins, sdk.NewCoin(r.Denom, r.Amount.TruncateInt()))
			}
			err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, coins)
			if err != nil {
				logger.Error("[Distribution] burn tokens", "error", err.Error())
				panic("bank keeper burn reward coins failed")
			}
			logger.Info("[Distribution] burn tokens successful", "validator", validator.GetOperator().String(), "reward", reward.String())
			return
		}
	}
	k.AllocateTokensToValidator(ctx, validator, reward)
}

// AllocateTokensToValidator allocate tokens to a particular validator,
// splitting according to commission.
func (k Keeper) AllocateTokensToValidator(ctx sdk.Context, val stakingtypes.ValidatorI, tokens sdk.DecCoins) {
	logger := ctx.Logger()
	logger.Debug("[Distribution] allocate tokens", "validator", val.GetOperator().String(), "reward", tokens.String())

	// update current rewards
	currentRewards := k.GetValidatorCurrentRewards(ctx, val.GetOperator())
	currentRewards.Rewards = currentRewards.Rewards.Add(tokens...)
	k.SetValidatorCurrentRewards(ctx, val.GetOperator(), currentRewards)

	// update outstanding rewards
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRewards,
			sdk.NewAttribute(sdk.AttributeKeyAmount, tokens.String()),
			sdk.NewAttribute(types.AttributeKeyValidator, val.GetOperator().String()),
		),
	)

	outstanding := k.GetValidatorOutstandingRewards(ctx, val.GetOperator())
	outstanding.Rewards = outstanding.Rewards.Add(tokens...)
	k.SetValidatorOutstandingRewards(ctx, val.GetOperator(), outstanding)
}
