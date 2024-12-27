package engine

import "github.com/ethereum-optimism/optimism/op-service/eth"

type PayloadInvalidEvent struct {
	Envelope *eth.ExecutionPayloadEnvelope
	Err      error

	ParentEv string
}

func (ev PayloadInvalidEvent) String() string {
	return "payload-invalid"
}

func (ev PayloadInvalidEvent) Parent() string {
	return ev.ParentEv
}

func (eq *EngDeriver) onPayloadInvalid(ev PayloadInvalidEvent) {
	eq.log.Warn("Payload was invalid", "block", ev.Envelope.ExecutionPayload.ID(),
		"err", ev.Err, "timestamp", uint64(ev.Envelope.ExecutionPayload.Timestamp))
}
