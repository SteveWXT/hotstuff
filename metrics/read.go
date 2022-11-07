package metrics

import (
	"time"

	"github.com/relab/hotstuff/eventloop"
	"github.com/relab/hotstuff/logging"
	"github.com/relab/hotstuff/metrics/types"
	"github.com/relab/hotstuff/modules"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/relab/hotstuff/pubsub"
)

func init() {
	RegisterReplicaMetric("read-throughput", func() any {
		return &ReadThroughput{}
	})
}

// Throughput measures throughput in commits per second, and commands per second.
type ReadThroughput struct {
	metricsLogger Logger
	opts          *modules.Options

	readCount uint64
}

// InitModule gives the module access to the other modules.
func (t *ReadThroughput) InitModule(mods *modules.Core) {
	var (
		eventLoop *eventloop.EventLoop
		logger    logging.Logger
	)

	mods.Get(
		&t.metricsLogger,
		&t.opts,
		&eventLoop,
		&logger,
	)

	eventLoop.RegisterHandler(pubsub.ReadMeasurementEvent{}, func(event any) {
		t.recordRead()
	})

	eventLoop.RegisterObserver(types.TickEvent{}, func(event any) {
		t.tick(event.(types.TickEvent))
	})

	logger.Info("ReadThroughput metric enabled")
}

func (t *ReadThroughput) recordRead() {
	t.readCount = t.readCount + 1
}

func (t *ReadThroughput) tick(tick types.TickEvent) {
	now := time.Now()
	event := &types.ReadMeasurement{
		Event:    types.NewReplicaEvent(uint32(t.opts.ID()), now),
		Count:    t.readCount,
		Duration: durationpb.New(now.Sub(tick.LastTick)),
	}
	t.metricsLogger.Log(event)
	// reset count for next tick
	t.readCount = 0
}
