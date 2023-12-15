package types

type EvmEventType int

const (
	EvmEventStakingCreateValidator EvmEventType = 1 // create validator
)

type EvmEvent struct {
	Type EvmEventType
	Data interface{}
}

type EvmEventCallback func(e *EvmEvent) error
