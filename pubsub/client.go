package pubsub

import (
	"github.com/SteveWXT/pubsub/clients"
	"github.com/relab/hotstuff/eventloop"
	"github.com/relab/hotstuff/logging"
	"github.com/relab/hotstuff/modules"
)

type PubSubClients struct {
	conns     []*clients.TCP
	logger    logging.Logger
	eventLoop *eventloop.EventLoop
}

func (c *PubSubClients) InitModule(mods *modules.Core) {
	mods.Get(
		&c.logger,
		&c.eventLoop,
	)
}

func NewPubSubClients() *PubSubClients {
	pbclients := &PubSubClients{
		conns: make([]*clients.TCP, 0),
	}
	return pbclients
}

// RunPubSubClients start a specific number of clients
func (c *PubSubClients) RunPubSubClients(host string, n int) {
	for i := 0; i < n; i++ {
		go c.addSubscriber(host)
	}
}

// addSubscriber start a client as subscriber
func (c *PubSubClients) addSubscriber(host string) {
	client, err := clients.New(host)
	if err != nil {
		c.logger.Fatalf("add subscriber error: %s", err)
	}

	c.conns = append(c.conns, client)
	client.Subscribe([]string{"topic"})

	ch := client.Messages()

	for msg := range ch {
		c.fakeProcess(msg.Data)
		c.eventLoop.AddEvent(ReadMeasurementEvent{})
	}

}

// fakeProcess used to simulate the client process messages
func (c *PubSubClients) fakeProcess(data string) {
	// c.logger.Debugf("pubsub client get %s", data)
}

type ReadMeasurementEvent struct {
}
