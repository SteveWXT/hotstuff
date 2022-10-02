package leaderrotation

import (
	"fmt"

	"github.com/relab/hotstuff"
	"github.com/relab/hotstuff/logging"
	"github.com/relab/hotstuff/modules"
	"golang.org/x/exp/slices"
)

func init() {
	modules.RegisterModule("CWA", NewCWA)
}

type repMap map[hotstuff.ID]float64

type cwa struct {
	configuration  modules.Configuration
	consensus      modules.Consensus
	opts           *modules.Options
	logger         logging.Logger
	blockChain     modules.BlockChain
	calculatedHead *hotstuff.Block
	reputations    repMap // latest reputations
}

type stack struct {
	list []*hotstuff.Block
}

func NewStack() *stack {
	return &stack{
		list: make([]*hotstuff.Block, 0),
	}
}

//入栈
func (s *stack) Push(x *hotstuff.Block) {
	s.list = append(s.list, x)
}

//出栈
func (s *stack) Pop() *hotstuff.Block {
	if len(s.list) <= 0 {
		fmt.Println("Stack is Empty")
		return nil
	} else {
		ret := s.list[len(s.list)-1]
		s.list = s.list[:len(s.list)-1]
		return ret
	}
}

func (s *stack) IsEmpty() bool {
	if len(s.list) == 0 {
		return true
	} else {
		return false
	}
}

var (
	maxRep     = 100.0
	proposeRep = 10.0
	voteRep    = 5.0
	decayPara  = 0.5
	delayPara  = 5
)

// InitModule gives the module a reference to the Core object.
// It also allows the module to set module options using the OptionsBuilder
func (c *cwa) InitModule(mods *modules.Core) {
	mods.Get(
		&c.configuration,
		&c.consensus,
		&c.opts,
		&c.logger,
		&c.blockChain,
	)
}

// GetLeader returns the id of the leader in the given view
func (c *cwa) GetLeader(view hotstuff.View) hotstuff.ID {
	recentCommit := c.consensus.CommittedBlock()
	refView := view - hotstuff.View(c.consensus.ChainLength()+delayPara)

	numReplicas := c.configuration.Len()

	if recentCommit.View() < refView {
		c.logger.Debugf("fallback to round-robin (view: %d, recent committed block: %d, but want ref view: %d)", view, recentCommit.View(), refView)
		return chooseRoundRobin(view, numReplicas)
	}

	// looking for reference block
	refBlock := recentCommit
	for refBlock.View() > refView && refBlock != hotstuff.GetGenesis() {
		refBlock, _ = c.blockChain.Get(refBlock.Parent())
	}

	// use round-robin for the first few views until we get a signature
	if refBlock.QuorumCert().Signature() == nil {
		c.logger.Debug("in startup; using round-robin")
		return chooseRoundRobin(view, numReplicas)
	}

	c.logger.Debugf("proceeding with CWA (looking leader of view: %d, recent committed block: %d, ref block view: %d)", view, recentCommit.View(), refBlock.View())

	var (
		f             = hotstuff.NumFaulty(c.configuration.Len())
		candidates    = make([]hotstuff.ID, 0, c.configuration.Len()-f)
		lastProposers = hotstuff.NewIDSet()

		i  = 0
		ok = true
	)

	// add reputations to proposer and voters of uncalculated blocks
	if c.calculatedHead.View() < refBlock.View() {

		var (
			ptr = refBlock
			stk = NewStack()
		)

		for ptr != c.calculatedHead {
			stk.Push(ptr)
			ptr, _ = c.blockChain.Get(ptr.Parent())
		}

		for !stk.IsEmpty() {

			prevBlock := stk.Pop()
			proposer := prevBlock.Proposer()

			// c.logger.Debugf("Calculate block %s: proposed by %s, voted by %s", prevBlock, proposer, voters)

			// reputation decays over time
			for k := range c.reputations {
				c.reputations[k] = decayPara * c.reputations[k]
			}

			c.reputations[proposer] += proposeRep
			if c.reputations[proposer] > maxRep {
				c.reputations[proposer] = maxRep
			}

			if prevBlock.QuorumCert().Signature() != nil {
				voters := prevBlock.QuorumCert().Signature().Participants()
				c.logger.Debugf("Voters: %v", voters)

				voters.ForEach(func(id hotstuff.ID) {
					c.reputations[id] += voteRep
					if c.reputations[id] > maxRep {
						c.reputations[id] = maxRep
					}
				})
			}
		}

		c.calculatedHead = refBlock
	}

	// debug info
	for id, score := range c.reputations {
		c.logger.Debugf("Current repuatation of %v: %v", id, score)
	}

	// get candidates of the next leader
	refBlockVoters := refBlock.QuorumCert().Signature().Participants()
	ptr := refBlock

	for ok && i < f && ptr != hotstuff.GetGenesis() {
		lastProposers.Add(ptr.Proposer())
		ptr, ok = c.blockChain.Get(ptr.Parent())
		i++
	}

	refBlockVoters.ForEach(func(id hotstuff.ID) {
		if !lastProposers.Contains(id) {
			candidates = append(candidates, id)
		}
	})

	// choose the next leader
	slices.SortFunc(candidates, func(a, b hotstuff.ID) bool {
		if c.reputations[a] == c.reputations[b] {
			return a < b
		} else {
			return c.reputations[a] < c.reputations[b]
		}
	})
	leader := candidates[len(candidates)-1]
	c.logger.Debugf("chose id %d (reputation: %v) for view %d", leader, c.reputations[leader], view)

	return leader
}

// NewRepBased returns a new random reputation-based leader rotation implementation
func NewCWA() modules.LeaderRotation {
	return &cwa{
		calculatedHead: hotstuff.GetGenesis(),
		reputations:    make(repMap),
	}
}
