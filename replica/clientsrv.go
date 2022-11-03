package replica

import (
	"crypto/sha256"
	"hash"
	"net"
	"sync"
	"time"

	"github.com/relab/hotstuff"

	"github.com/relab/gorums"
	"github.com/relab/hotstuff/eventloop"
	"github.com/relab/hotstuff/internal/proto/clientpb"
	"github.com/relab/hotstuff/logging"
	"github.com/relab/hotstuff/modules"
	"github.com/relab/hotstuff/pubsub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

// clientSrv serves a client.
type clientSrv struct {
	eventLoop *eventloop.EventLoop
	logger    logging.Logger

	mut          sync.Mutex
	srv          *gorums.Server
	awaitingCmds map[cmdID]chan<- error
	cmdCache     *cmdCache
	hash         hash.Hash

	connector *pubsub.ClientConn
	pbls      net.Listener
}

// newClientServer returns a new client server.
func newClientServer(conf Config, srvOpts []gorums.ServerOption) (srv *clientSrv) {
	srv = &clientSrv{
		awaitingCmds: make(map[cmdID]chan<- error),
		srv:          gorums.NewServer(srvOpts...),
		cmdCache:     newCmdCache(int(conf.BatchSize)),
		hash:         sha256.New(),
	}
	clientpb.RegisterClientServer(srv.srv, srv)
	return srv
}

// InitModule gives the module access to the other modules.
func (srv *clientSrv) InitModule(mods *modules.Core) {
	mods.Get(
		&srv.eventLoop,
		&srv.logger,
	)
	srv.cmdCache.InitModule(mods)
}

func (srv *clientSrv) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	srv.StartOnListener(lis)
	return nil
}

func (srv *clientSrv) StartOnListener(lis net.Listener) {
	go func() {
		err := srv.srv.Serve(lis)
		if err != nil {
			srv.logger.Error(err)
		}
	}()
}

func (srv *clientSrv) Stop() {
	srv.srv.Stop()
	if srv.connector != nil {
		srv.connector.Close()
	}
}

func (srv *clientSrv) ExecCommand(ctx gorums.ServerCtx, cmd *clientpb.Command) (*emptypb.Empty, error) {
	id := cmdID{cmd.ClientID, cmd.SequenceNumber}

	c := make(chan error)
	srv.mut.Lock()
	srv.awaitingCmds[id] = c
	srv.mut.Unlock()

	srv.cmdCache.addCommand(cmd)
	ctx.Release()
	err := <-c
	return &emptypb.Empty{}, err
}

func (srv *clientSrv) Exec(cmd hotstuff.Command) {
	batch := new(clientpb.Batch)
	err := proto.UnmarshalOptions{AllowPartial: true}.Unmarshal([]byte(cmd), batch)
	if err != nil {
		srv.logger.Errorf("Failed to unmarshal command: %v", err)
		return
	}

	srv.eventLoop.AddEvent(hotstuff.CommitEvent{Commands: len(batch.GetCommands())})

	for _, cmd := range batch.GetCommands() {
		_, _ = srv.hash.Write(cmd.Data)

		// relay the cmd to the pubsub module
		go srv.RelayToPubSub(cmd.Data)

		srv.mut.Lock()
		id := cmdID{cmd.GetClientID(), cmd.GetSequenceNumber()}
		if done, ok := srv.awaitingCmds[id]; ok {
			done <- nil
			delete(srv.awaitingCmds, id)
		}
		srv.mut.Unlock()
	}

	srv.logger.Debugf("Hash: %.8x", srv.hash.Sum(nil))
}

func (srv *clientSrv) Fork(cmd hotstuff.Command) {
	batch := new(clientpb.Batch)
	err := proto.UnmarshalOptions{AllowPartial: true}.Unmarshal([]byte(cmd), batch)
	if err != nil {
		srv.logger.Errorf("Failed to unmarshal command: %v", err)
		return
	}

	for _, cmd := range batch.GetCommands() {
		srv.mut.Lock()
		id := cmdID{cmd.GetClientID(), cmd.GetSequenceNumber()}
		if done, ok := srv.awaitingCmds[id]; ok {
			done <- status.Error(codes.Aborted, "blockchain was forked")
			delete(srv.awaitingCmds, id)
		}
		srv.mut.Unlock()
	}
}

// AddConnector add a pubsub connector (client)
func (srv *clientSrv) AddConnector(pbls net.Listener) {
	connector, err := pubsub.NewClient(pbls.Addr().String())
	if err != nil {
		srv.logger.Fatalf("PubSub server connector start failed: %v", err)
	}

	// srv.pbls = pbls
	srv.connector = connector
	srv.logger.Info("clientSrv: add connector")
}

// RelayToPubSub replay data to pusub module
func (srv *clientSrv) RelayToPubSub(data []byte) {
	if srv.connector == nil {
		srv.AddConnector(srv.pbls)
	}

	dataStr := "hello"
	srv.logger.Debugf("PubSub connector start to handle msg: %v", dataStr)
	mockSSE()
	err := srv.connector.Publish([]string{"topic"}, dataStr)
	if err != nil {
		srv.logger.Fatalf("Connector publish error: %v", err)
	}
}

// mockSSE simulate the SSE matching time for testing
func mockSSE() {
	time.Sleep(time.Millisecond * 3)
}
