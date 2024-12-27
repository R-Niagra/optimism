package derive

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/event"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type DeriverIdleEvent struct {
	Origin eth.L1BlockRef

	ParentEv string
}

func (d DeriverIdleEvent) String() string {
	return "derivation-idle"
}

func (d DeriverIdleEvent) Parent() string {
	return d.ParentEv
}

type DeriverL1StatusEvent struct {
	Origin eth.L1BlockRef
	LastL2 eth.L2BlockRef

	ParentEv string
}

func (d DeriverL1StatusEvent) String() string {
	return "deriver-l1-status"
}

func (d DeriverL1StatusEvent) Parent() string {
	return d.ParentEv
}

type DeriverMoreEvent struct {
	ParentEv string
}

func (d DeriverMoreEvent) String() string {
	return "deriver-more"
}

func (d DeriverMoreEvent) Parent() string {
	return d.ParentEv
}

// ConfirmReceivedAttributesEvent signals that the derivation pipeline may generate new attributes.
// After emitting DerivedAttributesEvent, no new attributes will be generated until a confirmation of reception.
type ConfirmReceivedAttributesEvent struct {
	ParentEv string
}

func (d ConfirmReceivedAttributesEvent) String() string {
	return "confirm-received-attributes"
}

func (d ConfirmReceivedAttributesEvent) Parent() string {
	return d.ParentEv
}

type ConfirmPipelineResetEvent struct {
	ParentEv string
}

func (d ConfirmPipelineResetEvent) String() string {
	return "confirm-pipeline-reset"
}

func (d ConfirmPipelineResetEvent) Parent() string {
	return d.ParentEv
}

// DerivedAttributesEvent is emitted when new attributes are available to apply to the engine.
type DerivedAttributesEvent struct {
	Attributes *AttributesWithParent

	ParentEv string
}

func (ev DerivedAttributesEvent) String() string {
	return "derived-attributes"
}

func (ev DerivedAttributesEvent) Parent() string {
	return ev.ParentEv
}

type PipelineStepEvent struct {
	PendingSafe eth.L2BlockRef

	ParentEv string
}

func (ev PipelineStepEvent) String() string {
	return "pipeline-step"
}

func (ev PipelineStepEvent) Parent() string {
	return ev.ParentEv
}

// DepositsOnlyPayloadAttributesRequestEvent requests a deposits-only version of the attributes from
// the pipeline. It is sent by the engine deriver and received by the PipelineDeriver.
// This event got introduced with Holocene.
type DepositsOnlyPayloadAttributesRequestEvent struct {
	ParentBlock eth.BlockID
	DerivedFrom eth.L1BlockRef

	ParentEv string
}

func (ev DepositsOnlyPayloadAttributesRequestEvent) String() string {
	return "deposits-only-payload-attributes-request"
}

func (ev DepositsOnlyPayloadAttributesRequestEvent) Parent() string {
	return ev.ParentEv
}

type PipelineDeriver struct {
	pipeline *DerivationPipeline

	ctx context.Context

	emitter event.Emitter

	needAttributesConfirmation bool
}

func NewPipelineDeriver(ctx context.Context, pipeline *DerivationPipeline) *PipelineDeriver {
	return &PipelineDeriver{
		pipeline: pipeline,
		ctx:      ctx,
	}
}

func (d *PipelineDeriver) AttachEmitter(em event.Emitter) {
	d.emitter = em
}

func (d *PipelineDeriver) OnEvent(ev event.Event) bool {
	switch x := ev.(type) {
	case rollup.ResetEvent:
		d.pipeline.Reset()
	case PipelineStepEvent:
		// Don't generate attributes if there are already attributes in-flight
		if d.needAttributesConfirmation {
			d.pipeline.log.Debug("Previously sent attributes are unconfirmed to be received")
			return true
		}
		d.pipeline.log.Trace("Derivation pipeline step", "onto_origin", d.pipeline.Origin())
		preOrigin := d.pipeline.Origin()
		attrib, err := d.pipeline.Step(d.ctx, x.PendingSafe)
		postOrigin := d.pipeline.Origin()
		if preOrigin != postOrigin {
			d.emitter.Emit(DeriverL1StatusEvent{Origin: postOrigin, LastL2: x.PendingSafe, ParentEv: "pipelineStep"})
		}
		if err == io.EOF {
			d.pipeline.log.Debug("Derivation process went idle", "progress", d.pipeline.Origin(), "err", err)
			d.emitter.Emit(DeriverIdleEvent{Origin: d.pipeline.Origin(), ParentEv: "pipelineStep"})
		} else if err != nil && errors.Is(err, EngineELSyncing) {
			d.pipeline.log.Debug("Derivation process went idle because the engine is syncing", "progress", d.pipeline.Origin(), "err", err)
			d.emitter.Emit(DeriverIdleEvent{Origin: d.pipeline.Origin(), ParentEv: "pipelineStep"})
		} else if err != nil && errors.Is(err, ErrReset) {
			d.emitter.Emit(rollup.ResetEvent{Err: err, ParentEv: "pipelineStep"})
		} else if err != nil && errors.Is(err, ErrTemporary) {
			d.emitter.Emit(rollup.EngineTemporaryErrorEvent{Err: err, ParentEv: "pipelineStep"})
		} else if err != nil && errors.Is(err, ErrCritical) {
			d.emitter.Emit(rollup.CriticalErrorEvent{Err: err, ParentEv: "pipelineStep"})
		} else if err != nil && errors.Is(err, NotEnoughData) {
			// don't do a backoff for this error
			d.emitter.Emit(DeriverMoreEvent{ParentEv: "pipelineStep"})
		} else if err != nil {
			d.pipeline.log.Error("Derivation process error", "err", err)
			d.emitter.Emit(rollup.EngineTemporaryErrorEvent{Err: err, ParentEv: "pipelineStep"})
		} else {
			if attrib != nil {
				d.emitDerivedAttributesEvent(attrib, "pipelineStep")
			} else {
				d.emitter.Emit(DeriverMoreEvent{ParentEv: "pipelineStep"}) // continue with the next step if we can
			}
		}
	case ConfirmPipelineResetEvent:
		d.pipeline.ConfirmEngineReset()
	case ConfirmReceivedAttributesEvent:
		d.needAttributesConfirmation = false
	case DepositsOnlyPayloadAttributesRequestEvent:
		d.pipeline.log.Warn("Deriving deposits-only attributes", "origin", d.pipeline.Origin())
		attrib, err := d.pipeline.DepositsOnlyAttributes(x.ParentBlock, x.DerivedFrom)
		if err != nil {
			d.emitter.Emit(rollup.CriticalErrorEvent{Err: fmt.Errorf("deriving deposits-only attributes: %w", err)})
			return true
		}
		d.emitDerivedAttributesEvent(attrib, "depositsOnlyPayloadAttributesRequest")
	default:
		return false
	}
	return true
}

func (d *PipelineDeriver) emitDerivedAttributesEvent(attrib *AttributesWithParent, parent string) {
	d.needAttributesConfirmation = true
	d.emitter.Emit(DerivedAttributesEvent{Attributes: attrib, ParentEv: parent})
}
