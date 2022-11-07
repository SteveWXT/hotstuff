package pubsub

import (
	"context"

	"github.com/relab/hotstuff/eventloop"
	"github.com/relab/hotstuff/logging"
	"github.com/relab/hotstuff/modules"
)

type PubSubClients struct {
	Conns     []*ClientConn
	Logger    logging.Logger
	EventLoop *eventloop.EventLoop
}

func (c *PubSubClients) InitModule(mods *modules.Core) {
	mods.Get(
		&c.Logger,
		&c.EventLoop,
	)
}

func NewPubSubClients() *PubSubClients {
	pbclients := &PubSubClients{
		Conns: make([]*ClientConn, 0),
	}
	return pbclients
}

// RunPubSubClients start a specific number of clients
func (c *PubSubClients) RunPubSubClients(host string, n int, ctx context.Context) {
	for i := 0; i < n; i++ {
		go c.addSubscriber(host, ctx)
	}
}

func (c *PubSubClients) Close() {
	for _, conn := range c.Conns {
		if conn != nil {
			conn.Close()
		}
	}
}

// addSubscriber start a client as subscriber
func (c *PubSubClients) addSubscriber(host string, ctx context.Context) {
	client, err := NewClient(host, ctx)
	if err != nil {
		c.Logger.Fatalf("add subscriber error: %v", err)
	}

	c.Conns = append(c.Conns, client)
	client.Subscribe([]string{"mock_topic"})
	ch := client.Messages()

	c.Logger.Info("A subscriber started at %v", host)

loop:
	// for msg := range ch {
	// 	c.fakeProcess(msg.Data)
	// 	c.EventLoop.AddEvent(ReadMeasurementEvent{})
	// 	select {
	// 	case <-ctx.Done():
	// 		break loop
	// 	default:

	// 	}
	// }

	for {
		select {
		case <-ctx.Done():
			break loop
		case msg := <-ch:
			c.fakeProcess(msg.Data)
			// c.EventLoop.AddEvent(ReadMeasurementEvent{})
		}
	}

}

// fakeProcess used to simulate the client process messages
func (c *PubSubClients) fakeProcess(data string) {
	// c.logger.Debugf("pubsub client get %s", data)
}

type ReadMeasurementEvent struct {
}
