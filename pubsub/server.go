package pubsub

import (
	"net"
	"time"

	"github.com/SteveWXT/pubsub/clients"
	"github.com/SteveWXT/pubsub/server"
	"github.com/relab/hotstuff/logging"
	"github.com/relab/hotstuff/modules"
)

type PubSubServer struct {
	ls        net.Listener
	logger    logging.Logger
	connector *clients.TCP
	pbclients *PubSubClients
}

func (pbsrv *PubSubServer) InitModule(mods *modules.Core) {
	mods.Get(&pbsrv.logger)
	pbsrv.pbclients.InitModule(mods)
}

func NewServer() *PubSubServer {
	pbsrv := &PubSubServer{
		pbclients: NewPubSubClients(),
	}

	return pbsrv
}

// Start start pubsub server and start a specific number of clients
func (pbsrv *PubSubServer) Start(ls net.Listener) {
	go server.StartWithLS(ls)

	port, _ := server.GetPort(ls)
	pbsrv.logger.Infof("PubSub server listen on port: %d", port)
	pbsrv.ls = ls

	connector, err := clients.New(ls.Addr().String())
	if err != nil {
		pbsrv.logger.Fatalf("PubSub server connector start failed: %v", err)
	}
	pbsrv.connector = connector

	// add subscribers
	n := 5
	pbsrv.pbclients.RunPubSubClients(ls.Addr().String(), n)
	pbsrv.logger.Infof("Started %v PubSub mock subscribers", n)
}

// HandleMsg send messages to the pubsub server by using the connector
func (pbsrv *PubSubServer) HandleMsg(data []byte) {

	if pbsrv.ls == nil {
		pblis, err := net.Listen("tcp", ":0")
		if err != nil {
			pbsrv.logger.Error(err)
		}
		pbsrv.Start(pblis)
	}

	dataStr := "hello"
	pbsrv.logger.Debugf("PubSub connector start to handle msg: %v", dataStr)
	mockSSE()
	err := pbsrv.connector.Publish([]string{"topic"}, dataStr)
	if err != nil {
		pbsrv.logger.Fatalf("Connector publish error: %v", err)
	}
}

// mockSSE simulate the SSE matching time for testing
func mockSSE() {
	time.Sleep(time.Millisecond * 3)
}
