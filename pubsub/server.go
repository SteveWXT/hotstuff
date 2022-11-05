package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/relab/hotstuff/logging"
	"github.com/relab/hotstuff/modules"
)

type PubSubServer struct {
	Ls        net.Listener
	Logger    logging.Logger
	Pbclients *PubSubClients

	ctx    context.Context
	cancel context.CancelFunc
}

func (pbsrv *PubSubServer) InitModule(mods *modules.Core) {
	mods.Get(&pbsrv.Logger)
	pbsrv.Pbclients.InitModule(mods)
}

func NewServer() *PubSubServer {
	pbsrv := &PubSubServer{
		Pbclients: NewPubSubClients(),
	}

	return pbsrv
}

// Start start pubsub server and start a specific number of clients
func (pbsrv *PubSubServer) Start(ls net.Listener) {

	// ls, err := net.Listen("tcp", ":0")
	// if err != nil {
	// 	pbsrv.Logger.DPanicf("pbsrv - start tcp listener error: %w", err)
	// }

	pbsrv.Ls = ls

	pbsrv.ctx, pbsrv.cancel = context.WithCancel(context.Background())

	success := make(chan struct{})
	go pbsrv.startWithLS(ls, pbsrv.ctx, success)

	port, _ := getPort(ls)
	pbsrv.Logger.Infof("PubSub server listen on port: %d", port)

	// add subscribers
	<-success
	n := 2
	pbsrv.Pbclients.RunPubSubClients(ls.Addr().String(), n, pbsrv.ctx)
	pbsrv.Logger.Infof("Started %v PubSub mock subscribers", n)
}

func (pbsrv *PubSubServer) GetClient() *ClientConn {
	client, err := NewClient(pbsrv.Ls.Addr().String(), pbsrv.ctx)
	if err != nil {
		pbsrv.Logger.Debugf("pubsub-server: GetClient error: %v", err)
	}
	return client
}

func (pbsrv *PubSubServer) Close() {
	pbsrv.Logger.Info("PubSub: Begin Close")

	pbsrv.cancel()
	if pbsrv.Ls != nil {
		pbsrv.Ls.Close()
	}

	pbsrv.Pbclients.Close()
	pbsrv.Logger.Info("PubSub: End Close")
}

func (pbsrv *PubSubServer) startWithLS(ls net.Listener, ctx context.Context, success chan<- struct{}) {

	errChan := make(chan error, 1)

	go pbsrv.startTCPWithLS(ls, errChan, ctx)

	select {
	case err := <-errChan:
		pbsrv.Logger.DPanicf("pbsrv - startWithLS error: %w", err)
	case <-time.After(time.Second * time.Duration(1)):
		// no errors
		success <- struct{}{}
	}

loop:
	for {
		select {
		case err := <-errChan:
			pbsrv.Logger.DPanicf("Server error - %s", err.Error())
		case <-ctx.Done():
			break loop
		}
	}
}

func (pbsrv *PubSubServer) startTCPWithLS(ls net.Listener, errChan chan<- error, ctx context.Context) {

	// start continually listening for any incoming tcp connections (non-blocking)
	go func(pbls net.Listener, pbctx context.Context) {
	loop:
		for {

			// accept connections
			conn, err := pbls.Accept()
			if err != nil {
				pbsrv.Logger.Debugf("pbsrv - startTCPWithLS error: %w", err)
			}

			select {
			case <-ctx.Done():
				if conn != nil {
					conn.Close()
				}
				break loop
			default:

			}

			// handle each connection individually (non-blocking)
			go pbsrv.handleConnectionWithCtx(conn, errChan, pbctx)
		}

	}(ls, ctx)

}

func (pbsrv *PubSubServer) handleConnectionWithCtx(conn net.Conn, errChan chan<- error, ctx context.Context) {

	// close the connection when we're done here
	defer conn.Close()

	// create a new client for each connection
	proxy := NewProxy()
	defer proxy.Close()

	handlers := GenerateHandlers()

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	go func(ctx context.Context) {
	loop:
		for msg := range proxy.Pipe {

			if err := encoder.Encode(msg); err != nil {
				pbsrv.Logger.Debugf("pbsrv - startTCPWithLS error: %w", err)
				break
			}
			select {
			case <-ctx.Done():
				break loop
			default:

			}
		}
	}(ctx)

loop:
	for {
		msg := Message{}

		if err := decoder.Decode(&msg); err != nil {
			switch err {
			case io.EOF:
				pbsrv.Logger.Debug("Client disconnected")
			case io.ErrUnexpectedEOF:
				pbsrv.Logger.Debug("Client disconnected unexpedtedly")
			default:
				errChan <- fmt.Errorf("Failed to decode message from TCP connection - %s", err.Error())
			}
			return
		}

		// look for the command
		handler, found := handlers[msg.Command]

		// if the command isn't found, return an error and wait for the next command
		if !found {
			pbsrv.Logger.Infof("Command '%s' not found", msg.Command)
			encoder.Encode(&Message{Command: msg.Command, Tags: msg.Tags, Data: msg.Data, Error: "Unknown Command"})
			continue
		}

		if err := handler(proxy, msg); err != nil {
			pbsrv.Logger.Debugf("TCP Failed to run '%s' - %s", msg.Command, err.Error())
			encoder.Encode(&Message{Command: msg.Command, Error: err.Error()})
			continue
		}

		select {
		case <-ctx.Done():
			break loop
		default:

		}
	}
}

func getPort(lis net.Listener) (uint32, error) {
	_, portStr, err := net.SplitHostPort(lis.Addr().String())
	if err != nil {
		return 0, err
	}
	port, err := strconv.ParseUint(portStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(port), nil
}
