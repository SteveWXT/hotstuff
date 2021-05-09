// Code generated by protoc-gen-gorums. DO NOT EDIT.
// versions:
// 	protoc-gen-gorums v0.4.0-devel
// 	protoc            v3.15.8
// source: client/client.proto

package client

import (
	context "context"
	fmt "fmt"
	gorums "github.com/relab/gorums"
	encoding "google.golang.org/grpc/encoding"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = gorums.EnforceVersion(4 - gorums.MinVersion)
	// Verify that the gorums runtime is sufficiently up-to-date.
	_ = gorums.EnforceVersion(gorums.MaxVersion - 4)
)

// A Configuration represents a static set of nodes on which quorum remote
// procedure calls may be invoked.
type Configuration struct {
	gorums.Configuration
	qspec QuorumSpec
}

// Nodes returns a slice of each available node. IDs are returned in the same
// order as they were provided in the creation of the Manager.
func (c *Configuration) Nodes() []*Node {
	nodes := make([]*Node, 0, c.Size())
	for _, n := range c.Configuration {
		nodes = append(nodes, &Node{n})
	}
	return nodes
}

// And returns a NodeListOption that can be used to create a new configuration combining c and d.
func (c Configuration) And(d *Configuration) gorums.NodeListOption {
	return c.Configuration.And(d.Configuration)
}

// Except returns a NodeListOption that can be used to create a new configuration
// from c without the nodes in rm.
func (c Configuration) Except(rm *Configuration) gorums.NodeListOption {
	return c.Configuration.Except(rm.Configuration)
}

func init() {
	if encoding.GetCodec(gorums.ContentSubtype) == nil {
		encoding.RegisterCodec(gorums.NewCodec())
	}
}

// Manager maintains a connection pool of nodes on
// which quorum calls can be performed.
type Manager struct {
	*gorums.Manager
}

// NewManager returns a new Manager for managing connection to nodes added
// to the manager. This function accepts manager options used to configure
// various aspects of the manager.
func NewManager(opts ...gorums.ManagerOption) (mgr *Manager) {
	mgr = &Manager{}
	mgr.Manager = gorums.NewManager(opts...)
	return mgr
}

// NewConfiguration returns a configuration based on the provided list of nodes (required)
// and an optional quorum specification. The QuorumSpec is necessary for call types that
// must process replies. For configurations only used for unicast or multicast call types,
// a QuorumSpec is not needed. The QuorumSpec interface is also a ConfigOption.
// Nodes can be supplied using WithNodeMap or WithNodeList, or WithNodeIDs.
// A new configuration can also be created from an existing configuration,
// using the And, WithNewNodes, Except, and WithoutNodes methods.
func (m *Manager) NewConfiguration(opts ...gorums.ConfigOption) (c *Configuration, err error) {
	if len(opts) < 1 || len(opts) > 2 {
		return nil, fmt.Errorf("wrong number of options: %d", len(opts))
	}
	c = &Configuration{}
	for _, opt := range opts {
		switch v := opt.(type) {
		case gorums.NodeListOption:
			c.Configuration, err = gorums.NewConfiguration(m.Manager, v)
			if err != nil {
				return nil, err
			}
		case QuorumSpec:
			// Must be last since v may match QuorumSpec if it is interface{}
			c.qspec = v
		default:
			return nil, fmt.Errorf("unknown option type: %v", v)
		}
	}
	// return an error if the QuorumSpec interface is not empty and no implementation was provided.
	var test interface{} = struct{}{}
	if _, empty := test.(QuorumSpec); !empty && c.qspec == nil {
		return nil, fmt.Errorf("missing required QuorumSpec")
	}
	return c, nil
}

// Nodes returns a slice of available nodes on this manager.
// IDs are returned in the order they were added at creation of the manager.
func (m *Manager) Nodes() []*Node {
	gorumsNodes := m.Manager.Nodes()
	nodes := make([]*Node, 0, len(gorumsNodes))
	for _, n := range gorumsNodes {
		nodes = append(nodes, &Node{n})
	}
	return nodes
}

type Node struct {
	*gorums.Node
}

// ExecCommand sends a command to all replicas and waits for valid signatures
// from f+1 replicas
func (c *Configuration) ExecCommand(ctx context.Context, in *Command) *AsyncEmpty {
	cd := gorums.QuorumCallData{
		Message: in,
		Method:  "client.Client.ExecCommand",
	}
	cd.QuorumFunction = func(req protoreflect.ProtoMessage, replies map[uint32]protoreflect.ProtoMessage) (protoreflect.ProtoMessage, bool) {
		r := make(map[uint32]*emptypb.Empty, len(replies))
		for k, v := range replies {
			r[k] = v.(*emptypb.Empty)
		}
		return c.qspec.ExecCommandQF(req.(*Command), r)
	}

	fut := c.Configuration.AsyncCall(ctx, cd)
	return &AsyncEmpty{fut}
}

// QuorumSpec is the interface of quorum functions for Client.
type QuorumSpec interface {
	gorums.ConfigOption

	// ExecCommandQF is the quorum function for the ExecCommand
	// asynchronous quorum call method. The in parameter is the request object
	// supplied to the ExecCommand method at call time, and may or may not
	// be used by the quorum function. If the in parameter is not needed
	// you should implement your quorum function with '_ *Command'.
	ExecCommandQF(in *Command, replies map[uint32]*emptypb.Empty) (*emptypb.Empty, bool)
}

// Client is the server-side API for the Client Service
type Client interface {
	ExecCommand(context.Context, *Command, func(*emptypb.Empty, error))
}

func RegisterClientServer(srv *gorums.Server, impl Client) {
	srv.RegisterHandler("client.Client.ExecCommand", func(ctx context.Context, in *gorums.Message, finished chan<- *gorums.Message) {
		req := in.Message.(*Command)
		once := new(sync.Once)
		f := func(resp *emptypb.Empty, err error) {
			once.Do(func() {
				select {
				case finished <- gorums.WrapMessage(in.Metadata, resp, err):
				case <-ctx.Done():
				}
			})
		}
		impl.ExecCommand(ctx, req, f)
	})
}

type internalEmpty struct {
	nid   uint32
	reply *emptypb.Empty
	err   error
}

// AsyncEmpty is a async object for processing replies.
type AsyncEmpty struct {
	*gorums.Async
}

// Get returns the reply and any error associated with the called method.
// The method blocks until a reply or error is available.
func (f *AsyncEmpty) Get() (*emptypb.Empty, error) {
	resp, err := f.Async.Get()
	if err != nil {
		return nil, err
	}
	return resp.(*emptypb.Empty), err
}
