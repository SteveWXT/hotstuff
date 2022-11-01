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
	isClose   bool
}

func (pbsrv *PubSubServer) InitModule(mods *modules.Core) {
	mods.Get(&pbsrv.logger)
	pbsrv.pbclients.InitModule(mods)
}

func NewServer() *PubSubServer {
	pbsrv := &PubSubServer{
		pbclients: NewPubSubClients(),
		isClose:   false,
	}

	return pbsrv
}

// Start start pubsub server and start a specific number of clients
func (pbsrv *PubSubServer) Start() {
	// 添加pubsub专用listener
	pbsrv.ls, _ = net.Listen("tcp", ":0")

	err := server.StartWithLS(pbsrv.ls)
	if err != nil {
		pbsrv.logger.DPanicf("PubSub server start error: %v", err)
	}

	port, _ := server.GetPort(pbsrv.ls)
	pbsrv.logger.Infof("PubSub server listen on port: %d", port)

	connector, err := clients.New(pbsrv.ls.Addr().String())
	if err != nil {
		pbsrv.logger.Fatalf("PubSub server connector start failed: %v", err)
	}
	pbsrv.connector = connector

	// add subscribers
	n := 2
	go pbsrv.pbclients.RunPubSubClients(pbsrv.ls.Addr().String(), n)
	pbsrv.logger.Infof("Started %v PubSub mock subscribers", n)
}

// HandleMsg send messages to the pubsub server by using the connector
func (pbsrv *PubSubServer) HandleMsg(data []byte) {
	if pbsrv.ls == nil && pbsrv.isClose == false {
		pbsrv.Start()
	}

	if pbsrv.connector == nil && pbsrv.isClose == false {
		pbsrv.connector, _ = clients.New(pbsrv.ls.Addr().String())
	}

	dataStr := "hello"
	pbsrv.logger.Debugf("PubSub connector start to handle msg: %v", dataStr)
	mockSSE()
	err := pbsrv.connector.Publish([]string{"topic"}, dataStr)
	if err != nil {
		pbsrv.logger.Fatalf("Connector publish error: %v", err)
	}
}

func (pbsrv *PubSubServer) Close() {
	pbsrv.logger.Info("PubSub: Begin Close")
	pbsrv.isClose = true
	if pbsrv.ls != nil {
		pbsrv.ls.Close()
	}
	if pbsrv.connector != nil {
		pbsrv.connector.Close()
	}
	pbsrv.pbclients.Close()
	pbsrv.logger.Info("PubSub: End Close")
}

// mockSSE simulate the SSE matching time for testing
func mockSSE() {
	time.Sleep(time.Millisecond * 3)
}
