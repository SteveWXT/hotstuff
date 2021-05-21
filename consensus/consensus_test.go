package consensus_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/relab/hotstuff/consensus"
	"github.com/relab/hotstuff/internal/testutil"
	"github.com/relab/hotstuff/synchronizer"
)

// TestVote checks that a leader can collect votes on a proposal to form a QC
func TestVote(t *testing.T) {
	const n = 4
	ctrl := gomock.NewController(t)
	bl := testutil.CreateBuilders(t, ctrl, n)
	bl[0].Register(synchronizer.New(testutil.FixedTimeout(1000)))
	hl := bl.Build()
	hs := hl[0]

	ok := false
	hs.EventLoop().RegisterAsyncHandler(func(event interface{}) (consume bool) {
		ok = true
		return true
	}, consensus.NewViewMsg{})

	b := testutil.NewProposeMsg(
		consensus.GetGenesis().Hash(),
		consensus.NewQuorumCert(nil, 1, consensus.GetGenesis().Hash()),
		"test", 1, 1,
	)
	hs.BlockChain().Store(b.Block)

	for i, signer := range hl.Signers() {
		pc, err := signer.CreatePartialCert(b.Block)
		if err != nil {
			t.Fatalf("Failed to create partial certificate: %v", err)
		}
		hs.VotingMachine().OnVote(consensus.VoteMsg{ID: consensus.ID(i + 1), PartialCert: pc})
	}
	if !ok {
		t.Error("No new view event happened")
	}
}
