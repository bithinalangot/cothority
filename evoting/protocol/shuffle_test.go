package protocol

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dedis/onet"

	"github.com/dedis/cothority"
	"github.com/dedis/cothority/evoting/lib"
)

var shuffleServiceID onet.ServiceID

type shuffleService struct {
	*onet.ServiceProcessor
	election *lib.Election
}

func init() {
	new := func(ctx *onet.Context) (onet.Service, error) {
		return &shuffleService{ServiceProcessor: onet.NewServiceProcessor(ctx)}, nil
	}
	shuffleServiceID, _ = onet.RegisterNewService(NameShuffle, new)
}

func (s *shuffleService) NewProtocol(n *onet.TreeNodeInstance, c *onet.GenericConfig) (
	onet.ProtocolInstance, error) {

	switch n.ProtocolName() {
	case NameShuffle:
		instance, _ := NewShuffle(n)
		shuffle := instance.(*Shuffle)
		shuffle.Election = s.election

		return shuffle, nil
	default:
		return nil, errors.New("Unknown protocol")
	}
}

func TestShuffleProtocol(t *testing.T) {
	for _, nodes := range []int{3, 5, 7} {
		runShuffle(t, nodes)
	}
}

func runShuffle(t *testing.T, n int) {
	local := onet.NewLocalTest(cothority.Suite)
	defer local.CloseAll()

	nodes, roster, tree := local.GenBigTree(n, n, 1, true)

	election := &lib.Election{Roster: roster, Stage: lib.Running}
	_ = election.GenChain(n)

	services := local.GetServices(nodes, shuffleServiceID)
	for i := range services {
		services[i].(*shuffleService).election = election
	}

	instance, _ := services[0].(*shuffleService).CreateProtocol(NameShuffle, tree)
	shuffle := instance.(*Shuffle)
	shuffle.Election = election
	shuffle.Start()

	select {
	case <-shuffle.Finished:
		box, _ := election.Box()
		mixes, _ := election.Mixes()

		in1, in2 := lib.Split(box.Ballots)
		for i := range mixes {
			out1, out2 := lib.Split(mixes[i].Ballots)
			require.Nil(t, lib.Verify(mixes[i].Proof, election.Key, in1, in2, out1, out2))
			in1, in2 = out1, out2
		}
	case <-time.After(60 * time.Second):
		t.Fatal("Protocol timeout")
	}
}
