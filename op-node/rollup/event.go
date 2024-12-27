package rollup

import "github.com/ethereum-optimism/optimism/op-node/rollup/event"

// L1TemporaryErrorEvent identifies a temporary issue with the L1 data.
type L1TemporaryErrorEvent struct {
	Err      error
	ParentEv string
}

var _ event.Event = L1TemporaryErrorEvent{}

func (ev L1TemporaryErrorEvent) String() string {
	return "l1-temporary-error"
}

func (ev L1TemporaryErrorEvent) Parent() string {
	return ev.ParentEv
}

// EngineTemporaryErrorEvent identifies a temporary processing issue.
// It applies to both L1 and L2 data, often inter-related.
// This scope will be reduced over time, to only capture L2-engine specific temporary errors.
// See L1TemporaryErrorEvent for L1 related temporary errors.
type EngineTemporaryErrorEvent struct {
	Err      error
	ParentEv string
}

var _ event.Event = EngineTemporaryErrorEvent{}

func (ev EngineTemporaryErrorEvent) String() string {
	return "engine-temporary-error"
}

func (ev EngineTemporaryErrorEvent) Parent() string {
	return ev.ParentEv
}

type ResetEvent struct {
	Err      error
	ParentEv string
}

var _ event.Event = ResetEvent{}

func (ev ResetEvent) String() string {
	return "reset-event"
}

func (ev ResetEvent) Parent() string {
	return ev.ParentEv
}

// CriticalErrorEvent is an alias for event.CriticalErrorEvent
type CriticalErrorEvent = event.CriticalErrorEvent
