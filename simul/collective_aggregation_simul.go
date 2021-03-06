package main

import (
	"errors"
	"github.com/BurntSushi/toml"
	"github.com/ldsec/unlynx/lib"
	"github.com/ldsec/unlynx/lib/aggregation"
	"github.com/ldsec/unlynx/protocols"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
)

func createDataSet(numberGroups, numberAttributes, numberGroupAttr int) (map[libunlynx.GroupingKey]libunlynx.FilteredResponse, error) {
	var secContrib = libunlynx.SuiTe.Scalar().One()
	var clientPrivate = libunlynx.SuiTe.Scalar().One() //one -> to have the same for each node
	var clientPublic = libunlynx.SuiTe.Point().Mul(clientPrivate, libunlynx.SuiTe.Point().Base())

	testCVMap := make(map[libunlynx.GroupingKey]libunlynx.FilteredResponse)

	tabGrp := make([]int64, numberGroupAttr)
	for i := 0; i < numberGroupAttr; i++ {
		tabGrp[i] = int64(1)
	}

	dummyGroups := *libunlynx.EncryptIntVector(clientPublic, tabGrp)
	for i := 0; i < numberGroups; i++ {
		tab := make([]int64, numberAttributes)
		for i := 0; i < numberAttributes; i++ {
			tab[i] = int64(1)
		}

		cipherVect := *libunlynx.EncryptIntVector(clientPublic, tab)

		tag, err := protocolsunlynx.CipherVectorToDeterministicTag(*libunlynx.EncryptIntVector(clientPublic, []int64{int64(i)}), clientPrivate, secContrib, clientPublic, false)
		if err != nil {
			return nil, err
		}
		testCVMap[tag] = libunlynx.FilteredResponse{GroupByEnc: dummyGroups, AggregatingAttributes: cipherVect}
	}
	return testCVMap, nil
}

func init() {
	onet.SimulationRegister("CollectiveAggregation", NewCollectiveAggregationSimulation)
}

// CollectiveAggregationSimulation hogithub.com/ldsec/ the state of a simulation.
type CollectiveAggregationSimulation struct {
	onet.SimulationBFTree

	NbrGroups          int
	NbrGroupAttributes int
	NbrAggrAttributes  int
	Proofs             bool
}

// NewCollectiveAggregationSimulation is the simulation instance constructor.
func NewCollectiveAggregationSimulation(config string) (onet.Simulation, error) {
	sim := &CollectiveAggregationSimulation{}
	_, err := toml.Decode(config, sim)
	if err != nil {
		return nil, err
	}

	return sim, nil
}

// Setup initializes the simulation.
func (sim *CollectiveAggregationSimulation) Setup(dir string, hosts []string) (*onet.SimulationConfig, error) {
	sc := &onet.SimulationConfig{}
	sim.CreateRoster(sc, hosts, 2000)
	err := sim.CreateTree(sc)

	if err != nil {
		return nil, err
	}

	log.Lvl1("Setup done")

	return sc, nil
}

// Node registers a CollectiveAggregationSimul (with access to the CollectiveAggregationSimulation object) for every node
func (sim *CollectiveAggregationSimulation) Node(config *onet.SimulationConfig) error {
	if _, err := config.Server.ProtocolRegister("CollectiveAggregationSimul",
		func(tni *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
			return NewAggregationProtocolSimul(tni, sim)
		}); err != nil {
		return errors.New("Error while registering <CollectiveAggregationSimul>:" + err.Error())
	}

	return sim.SimulationBFTree.Node(config)
}

// Run starts the simulation of the protocol and measures its runtime.
func (sim *CollectiveAggregationSimulation) Run(config *onet.SimulationConfig) error {
	for round := 0; round < sim.Rounds; round++ {
		log.Lvl1("Starting round", round)
		rooti, err := config.Overlay.CreateProtocol("CollectiveAggregationSimul", config.Tree, onet.NilServiceID)
		if err != nil {
			log.Lvl1("error Run")
			return err
		}

		root := rooti.(*protocolsunlynx.CollectiveAggregationProtocol)

		//time measurement
		round := libunlynx.StartTimer("CollectiveAggregation(SIMULATION)")

		log.Lvl1("Start protocol")
		if err := root.Start(); err != nil {
			return err
		}
		<-root.ProtocolInstance().(*protocolsunlynx.CollectiveAggregationProtocol).FeedbackChannel

		libunlynx.EndTimer(round)
	}

	return nil
}

// NewAggregationProtocolSimul is a simulation specific protocol instance constructor that injects test data.
func NewAggregationProtocolSimul(tni *onet.TreeNodeInstance, sim *CollectiveAggregationSimulation) (onet.ProtocolInstance, error) {
	protocol, err := protocolsunlynx.NewCollectiveAggregationProtocol(tni)
	collectiveAggr := protocol.(*protocolsunlynx.CollectiveAggregationProtocol)

	data, err := createDataSet(sim.NbrGroups, sim.NbrAggrAttributes, sim.NbrGroupAttributes)
	if err != nil {
		return nil, err
	}
	collectiveAggr.GroupedData = &data
	collectiveAggr.Proofs = sim.Proofs
	collectiveAggr.ProofFunc = func(data []libunlynx.CipherVector, res libunlynx.CipherVector) *libunlynxaggr.PublishedAggregationListProof {
		proof := libunlynxaggr.AggregationListProofCreation(data, res)
		return &proof
	}

	return collectiveAggr, err
}
