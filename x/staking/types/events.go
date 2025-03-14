package types

// staking module event types
const (
	EventTypeCompleteUnbonding         = "complete_unbonding"
	EventTypeCompleteRedelegation      = "complete_redelegation"
	EventTypeCreateValidator           = "create_validator"
	EventTypeEditValidator             = "edit_validator"
	EventTypeDelegate                  = "delegate"
	EventTypeUnbond                    = "unbond"
	EventTypeCandidateUnbond           = "candidate_unbond"
	EventTypeCancelUnbondingDelegation = "cancel_unbonding_delegation"
	EventTypeRedelegate                = "redelegate"
	EventTypeValidatorDelegate         = "validator_delegate"
	AttributeKeyValidator              = "validator"
	AttributeKeyCommissionRate         = "commission_rate"
	AttributeKeyMinSelfDelegation      = "min_self_delegation"
	AttributeKeySrcValidator           = "source_validator"
	AttributeKeyDstValidator           = "destination_validator"
	AttributeKeyDelegator              = "delegator"
	AttributeKeyCreationHeight         = "creation_height"
	AttributeKeyCompletionTime         = "completion_time"
	AttributeKeyNewShares              = "new_shares"
)
