package hotstuff_test

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/relab/hotstuff"
	"github.com/relab/hotstuff/backend/gorums"
	"github.com/relab/hotstuff/config"
	"github.com/relab/hotstuff/consensus/chainedhotstuff"
	"github.com/relab/hotstuff/internal/mocks"
	"github.com/relab/hotstuff/internal/testutil"
	"github.com/relab/hotstuff/synchronizer"
)

// TestChainedHotstuff runs chained hotstuff with the gorums backend and expects each replica to execute 10 times.
func TestChainedHotstuff(t *testing.T) {
	const n = 4
	ctrl := gomock.NewController(t)

	baseCfg := config.NewConfig(0, nil, nil)

	listeners := make([]net.Listener, n)
	keys := make([]hotstuff.PrivateKey, n)
	for i := 0; i < n; i++ {
		listeners[i] = testutil.CreateTCPListener(t)
		key := testutil.GenerateECDSAKey(t)
		keys[i] = key
		id := hotstuff.ID(i + 1)
		baseCfg.Replicas[id] = &config.ReplicaInfo{
			ID:      id,
			Address: listeners[i].Addr().String(),
			PubKey:  key.Public(),
		}
	}

	builders := testutil.CreateBuilders(t, ctrl, n, keys...)
	configs := make([]*gorums.Config, n)
	servers := make([]*gorums.Server, n)
	synchronizers := make([]hotstuff.ViewSynchronizer, n)
	for i := 0; i < n; i++ {
		c := *baseCfg
		c.ID = hotstuff.ID(i + 1)
		c.PrivateKey = keys[i].(*ecdsa.PrivateKey)
		configs[i] = gorums.NewConfig(c)
		servers[i] = gorums.NewServer(c)
		synchronizers[i] = synchronizer.New(
			hotstuff.ExponentialTimeout{Base: 100 * time.Millisecond, ExponentBase: 2, MaxExponent: 10},
		)
		builders[i].Register(chainedhotstuff.New(), configs[i], servers[i], synchronizers[i])
	}

	executors := make([]*mocks.MockExecutor, n)
	counters := make([]uint, n)
	c := make(chan struct{}, n)
	errChan := make(chan error, n)
	for i := 0; i < n; i++ {
		counter := &counters[i]
		executors[i] = mocks.NewMockExecutor(ctrl)
		executors[i].EXPECT().Exec(gomock.Any()).AnyTimes().Do(func(arg hotstuff.Command) {
			if arg != hotstuff.Command("foo") {
				errChan <- fmt.Errorf("unknown command executed: got %s, want: %s", arg, "foo")
			}
			*counter++
			if *counter >= 100 {
				c <- struct{}{}
			}
		})
		builders[i].Register(executors[i])
	}

	hl := builders.Build()

	ctx, cancel := context.WithCancel(context.Background())

	for i, server := range servers {
		server.StartOnListener(listeners[i])
		defer server.Stop()
	}

	for _, cfg := range configs {
		err := cfg.Connect(time.Second)
		if err != nil {
			t.Fatal(err)
		}
		defer cfg.Close()
	}

	for _, hs := range hl {
		go hs.EventLoop().Run(ctx)
	}

	for i := 0; i < n; i++ {
		select {
		case <-c:
			defer synchronizers[i].Stop()
		case err := <-errChan:
			t.Fatal(err)
		}
	}
	cancel()
}
