package types

type GovEventType int

const (
	GovEventCheckValidatorStatus GovEventType = 1 // check validator status
	GovEventSetValidatorStatus   GovEventType = 2 // set validator status
)

type GovEvent struct {
	Type GovEventType
	Data interface{}
}

type GovEventCallback func(ctx Context, e *GovEvent) error
