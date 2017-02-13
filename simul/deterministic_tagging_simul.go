package main

import (
	"github.com/BurntSushi/toml"
	"github.com/JoaoAndreSa/MedCo/lib"
	"github.com/JoaoAndreSa/MedCo/protocols"
	"gopkg.in/dedis/crypto.v0/random"
	"gopkg.in/dedis/onet.v1"
	"gopkg.in/dedis/onet.v1/log"
	"gopkg.in/dedis/onet.v1/network"
	"gopkg.in/dedis/onet.v1/simul/monitor"
)

var groupingAttr int
var aggrAttr int
var k int
var proofs bool

func init() {
	onet.SimulationRegister("DeterministicTagging", NewDeterministicTaggingSimulation)
	onet.GlobalProtocolRegister("DeterministicTaggingSimul", NewDeterministicTaggingSimul)

}

// DeterministicTaggingSimulation is the structure holding the state of the simulation.
type DeterministicTaggingSimulation struct {
	onet.SimulationBFTree

	NbrResponses       int
	NbrGroupAttributes int
	NbrAggrAttributes  int
	Proofs             bool
}

// NewDeterministicTaggingSimulation is a constructor for the simulation.
func NewDeterministicTaggingSimulation(config string) (onet.Simulation, error) {
	sim := &DeterministicTaggingSimulation{}
	_, err := toml.Decode(config, sim)

	if err != nil {
		return nil, err
	}
	return sim, nil
}

// Setup initializes a simulation.
func (sim *DeterministicTaggingSimulation) Setup(dir string, hosts []string) (*onet.SimulationConfig, error) {
	sc := &onet.SimulationConfig{}
	sim.CreateRoster(sc, hosts, 20)
	err := sim.CreateTree(sc)

	if err != nil {
		return nil, err
	}

	log.Lvl1("Setup done")

	return sc, nil
}

// Run starts the simulation.
func (sim *DeterministicTaggingSimulation) Run(config *onet.SimulationConfig) error {
	groupingAttr = sim.NbrGroupAttributes
	aggrAttr = sim.NbrAggrAttributes
	k = sim.NbrResponses
	proofs = sim.Proofs
	for round := 0; round < sim.Rounds; round++ {
		log.Lvl1("Starting round", round)
		rooti, err := config.Overlay.CreateProtocol("DeterministicTaggingSimul", config.Tree, onet.NilServiceID)

		if err != nil {
			return err
		}

		root := rooti.(*protocols.DeterministicTaggingProtocol)

		//complete protocol time measurement
		round := monitor.NewTimeMeasure("DetTagging(SIMULATION)")
		root.Start()

		<-root.ProtocolInstance().(*protocols.DeterministicTaggingProtocol).FeedbackChannel

		round.Record()
	}

	return nil
}

// NewDeterministicTaggingSimul is a custom protocol constructor specific for simulation purposes.
func NewDeterministicTaggingSimul(tni *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	protocol, err := protocols.NewDeterministicTaggingProtocol(tni)
	pap := protocol.(*protocols.DeterministicTaggingProtocol)
	pap.Proofs = proofs
	if tni.IsRoot() {
		aggregateKey := pap.Roster().Aggregate

		// Creates dummy data...
		clientResponses := make([]lib.ClientResponse, k)
		tabGroup := make([]int64, groupingAttr)
		tabAttr := make([]int64, aggrAttr)

		for i := 0; i < groupingAttr; i++ {
			tabGroup[i] = int64(1)
		}
		for i := 0; i < aggrAttr; i++ {
			tabAttr[i] = int64(1)
		}

		encryptedGrp := *lib.EncryptIntVector(aggregateKey, tabGroup)
		encryptedAttr := *lib.EncryptIntVector(aggregateKey, tabAttr)
		clientResponse := lib.ClientResponse{ProbaGroupingAttributesEnc: encryptedGrp, AggregatingAttributes: encryptedAttr}

		for i := 0; i < k; i++ {
			clientResponses[i] = clientResponse
		}

		pap.TargetOfSwitch = &clientResponses
	}
	tempKey := network.Suite.Scalar().Pick(random.Stream)
	pap.SurveySecretKey = &tempKey

	return pap, err
}