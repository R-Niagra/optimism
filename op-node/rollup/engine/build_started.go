package engine

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type BuildStartedEvent struct {
	Info eth.PayloadInfo

	BuildStarted time.Time

	ParentBlock eth.L2BlockRef

	// if payload should be promoted to (local) safe (must also be pending safe, see DerivedFrom)
	Concluding bool
	// payload is promoted to pending-safe if non-zero
	DerivedFrom eth.L1BlockRef

	ParentEv string
}

func (ev BuildStartedEvent) String() string {
	return "build-started"
}

func (ev BuildStartedEvent) Parent() string {
	return ev.ParentEv
}

func (eq *EngDeriver) onBuildStarted(ev BuildStartedEvent) {
	// If a (pending) safe block, immediately seal the block
	if ev.DerivedFrom != (eth.L1BlockRef{}) {
		eq.emitter.Emit(BuildSealEvent{
			Info:         ev.Info,
			BuildStarted: ev.BuildStarted,
			Concluding:   ev.Concluding,
			DerivedFrom:  ev.DerivedFrom,
		})
	}
}
