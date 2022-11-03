package pubsub

import (
	"context"

	"github.com/SteveWXT/pubsub/clients"
	"github.com/relab/hotstuff/eventloop"
	"github.com/relab/hotstuff/logging"
	"github.com/relab/hotstuff/modules"
)

type PubSubClients struct {
	Conns     []*clients.TCP
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
		Conns: make([]*clients.TCP, 0),
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
	client, err := clients.New(host)
	if err != nil {
		c.Logger.Fatalf("add subscriber error: %v", err)
	}

	c.Conns = append(c.Conns, client)
	client.Subscribe([]string{"topic"})
	ch := client.Messages()

	c.Logger.Info("A subscriber started at %v", host)

loop:
	for msg := range ch {
		c.fakeProcess(msg.Data)
		c.EventLoop.AddEvent(ReadMeasurementEvent{})
		select {
		case <-ctx.Done():
			break loop
		default:

		}
	}

}

// fakeProcess used to simulate the client process messages
func (c *PubSubClients) fakeProcess(data string) {
	// c.logger.Debugf("pubsub client get %s", data)
}

type ReadMeasurementEvent struct {
}
