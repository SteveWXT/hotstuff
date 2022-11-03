package pubsub

import (
	"context"
	"net"

	"github.com/SteveWXT/pubsub/server"
	"github.com/relab/hotstuff/logging"
	"github.com/relab/hotstuff/modules"
)

type PubSubServer struct {
	Ls        net.Listener
	Logger    logging.Logger
	Pbclients *PubSubClients
	IsClose   bool

	cancel context.CancelFunc
}

func (pbsrv *PubSubServer) InitModule(mods *modules.Core) {
	mods.Get(&pbsrv.Logger)
	pbsrv.Pbclients.InitModule(mods)
}

func NewServer() *PubSubServer {
	pbsrv := &PubSubServer{
		Pbclients: NewPubSubClients(),
		IsClose:   false,
	}

	return pbsrv
}

// Start start pubsub server and start a specific number of clients
func (pbsrv *PubSubServer) Start(ls net.Listener) {
	pbsrv.Ls = ls

	var ctx context.Context
	ctx, pbsrv.cancel = context.WithCancel(context.Background())
	go server.StartWithLS(ls, ctx)

	port, _ := server.GetPort(ls)
	pbsrv.Logger.Infof("PubSub server listen on port: %d", port)

	// add subscribers
	n := 2
	go pbsrv.Pbclients.RunPubSubClients(ls.Addr().String(), n, ctx)
	pbsrv.Logger.Infof("Started %v PubSub mock subscribers", n)
}

// HandleMsg send messages to the pubsub server by using the connector
// func (pbsrv *PubSubServer) HandleMsg(data []byte) {
// if pbsrv.ls == nil && pbsrv.isClose == false {
// 	// 添加pubsub专用listener
// 	pubsubListen, _ := net.Listen("tcp", ":0")
// 	pbsrv.Start(pubsubListen)
// }

// 	if pbsrv.connector == nil && pbsrv.isClose == false {
// 		pbsrv.connector, _ = clients.New(pbsrv.ls.Addr().String())
// 	}

// 	dataStr := "hello"
// 	pbsrv.logger.Debugf("PubSub connector start to handle msg: %v", dataStr)
// 	mockSSE()
// 	err := pbsrv.connector.Publish([]string{"topic"}, dataStr)
// 	if err != nil {
// 		pbsrv.logger.Fatalf("Connector publish error: %v", err)
// 	}
// }

func (pbsrv *PubSubServer) Close() {
	pbsrv.Logger.Info("PubSub: Begin Close")
	pbsrv.IsClose = true

	pbsrv.cancel()

	if pbsrv.Ls != nil {
		pbsrv.Ls.Close()
	}
	// if pbsrv.connector != nil {
	// 	pbsrv.connector.Close()
	// }
	pbsrv.Pbclients.Close()
	pbsrv.Logger.Info("PubSub: End Close")
}
