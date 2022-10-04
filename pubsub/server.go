package pubsub

import (
	"net"

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

func (pbsrv *PubSubServer) Start(ls net.Listener) {
	go server.StartWithLS(ls)

	port, _ := server.GetPort(ls)
	pbsrv.logger.Infof("PubSub server listen on port: %d", port)
	pbsrv.ls = ls

	connector, err := clients.New(ls.Addr().String())
	if err != nil {
		pbsrv.logger.Fatalf("PubSub server connector start failed: %s", err)
	}
	pbsrv.connector = connector

	// add subscribers
	pbsrv.pbclients.RunPubSubClients(ls.Addr().String(), 5)
}

func (pbsrv *PubSubServer) HandleMsg(data []byte) {
	temp := "hello"
	pbsrv.logger.Debugf("PubSub connector start to handle msg: %s", temp)
	pbsrv.connector.Publish([]string{"topic"}, temp)
}
