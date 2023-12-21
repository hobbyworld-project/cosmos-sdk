package types

type EvmEventType int

const (
	EvmEventCheckValidatorStatus EvmEventType = 1 // check validator status
	EvmEventSetValidatorStatus   EvmEventType = 2 // set validator status
)

type EvmEvent struct {
	Type EvmEventType
	Data interface{}
}

type EvmEventCallback func(ctx Context, e *EvmEvent) error
