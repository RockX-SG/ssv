package ssv

import (
	"encoding/hex"
	spec "github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/bloxapp/ssv/beacon"
	"github.com/bloxapp/ssv/docs/spec/qbft"
	"github.com/bloxapp/ssv/docs/spec/types"
	"github.com/bloxapp/ssv/ibft/proto"
	"github.com/pkg/errors"
)

var testingValidatorPK = spec.BLSPubKey{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4}
var committee = []*types.Node{
	{
		NodeID: 1,
	},
	{
		NodeID: 2,
	},
	{
		NodeID: 3,
	},
	{
		NodeID: 4,
	},
}

func newTestingValidator() *Validator {
	return &Validator{
		valCheck: &testingValCheckPassAll{},
	}
}

func newTestingDutyExecutionState() *dutyExecutionState {
	return &dutyExecutionState{
		collectedPartialSigs: make(map[types.NodeID][]byte),
	}
}

type testingQBFTController struct {
	instances  map[uint64]*testingQBFTInstance
	height     uint64
	identifier []byte
}

func newTestingQBFTController(identifier []byte) *testingQBFTController {
	return &testingQBFTController{
		identifier: identifier,
		height:     0,
		instances:  make(map[uint64]*testingQBFTInstance),
	}
}

// StartNewInstance will start a new QBFT instance, if can't will return error
func (tContr *testingQBFTController) StartNewInstance(value []byte) error {
	inst := newTestingQBFTInstance()
	tContr.height++
	inst.height = tContr.height
	tContr.instances[inst.height] = inst
	return nil
}

// ProcessMsg processes a new msg, returns true if decided, non nil byte slice if decided (decided value) and error
// decided returns just once per instance as true, following messages (for example additional commit msgs) will not return decided true
func (tContr *testingQBFTController) ProcessMsg(msg *qbft.SignedMessage) (bool, []byte, error) {
	return false, nil, nil
}

// InstanceForHeight returns an instance for a specific height, nil if not found
func (tContr *testingQBFTController) InstanceForHeight(height uint64) qbft.IInstance {
	if inst, found := tContr.instances[height]; found {
		return inst
	}
	return nil
}

// GetHeight returns the current running instance height or, if not started, the last decided height
func (tContr *testingQBFTController) GetHeight() uint64 {
	return tContr.height
}

// GetIdentifier returns QBFT identifier, used to identify messages
func (tContr *testingQBFTController) GetIdentifier() []byte {
	return tContr.identifier
}

type testingQBFTInstance struct {
	height  uint64
	decided bool
}

func newTestingQBFTInstance() *testingQBFTInstance {
	return &testingQBFTInstance{
		height:  1,
		decided: false,
	}
}

// Start implementation
func (tInstance *testingQBFTInstance) Start(value []byte, height uint64) {

}

// ProcessMsg implementation
func (tInstance *testingQBFTInstance) ProcessMsg(msg *qbft.SignedMessage) (decided bool, decidedValue []byte, err error) {
	return false, nil, nil

}

// IsDecided implementation
func (tInstance *testingQBFTInstance) IsDecided() (bool, []byte) {
	return tInstance.decided, nil
}

// GetHeight implementation
func (tInstance *testingQBFTInstance) GetHeight() uint64 {
	return tInstance.height
}

type testingStorage struct {
	storage map[string]map[beacon.RoleType]*consensusData
}

func newTestingStorage() *testingStorage {
	return &testingStorage{
		storage: make(map[string]map[beacon.RoleType]*consensusData),
	}
}

// SaveHighestDecided saves the decided value as highest for a validator PK and role
func (s *testingStorage) SaveHighestDecided(validatorPK []byte, role beacon.RoleType, decidedValue *consensusData) error {
	if s.storage[hex.EncodeToString(validatorPK)] == nil {
		s.storage[hex.EncodeToString(validatorPK)] = make(map[beacon.RoleType]*consensusData)
	}
	s.storage[hex.EncodeToString(validatorPK)][role] = decidedValue
	return nil
}

// GetHighestDecided returns the saved decided value (highest) for a validator PK and role
func (s *testingStorage) GetHighestDecided(validatorPK []byte, role beacon.RoleType) (*consensusData, error) {
	if s.storage[hex.EncodeToString(validatorPK)] == nil {
		return nil, errors.New("can't find validator PK")
	}
	if value, found := s.storage[hex.EncodeToString(validatorPK)][role]; found {
		return value, nil
	}
	return nil, errors.New("can't find role")
}

func newTestingDutyRunner() *DutyRunner {
	return &DutyRunner{
		beaconRoleType: beacon.RoleTypeAttester,
		validatorPK:    testingValidatorPK[:],
		storage:        newTestingStorage(),
		qbftController: newTestingQBFTController([]byte{1, 2, 3, 4}),
		nodeID:         1,
		share: &Share{
			pubKey:    testingValidatorPK[:],
			committee: committee,
			quorum:    3,
		},
	}
}

type testingSigner struct {
}

// SignIBFTMessage signs a network iBFT msg
func (s *testingSigner) SignIBFTMessage(message *proto.Message, pk []byte) ([]byte, error) {
	return nil, nil
}

// SignAttestation signs the given attestation
func (s *testingSigner) SignAttestation(data *spec.AttestationData, duty *beacon.Duty, pk []byte) (*spec.Attestation, []byte, error) {
	sig := spec.BLSSignature{1, 2, 3, 4}
	att := &spec.Attestation{
		Data:      data,
		Signature: sig,
	}
	return att, sig[:], nil
}

type testingNode struct {
	nodeID types.NodeID
	pk     []byte
}

func (n *testingNode) GetPublicKey() []byte {
	return n.pk
}

// GetID returns the node's ID
func (n *testingNode) GetID() types.NodeID {
	return n.nodeID
}

type testingValCheckPassAll struct {
}

func (valCheck *testingValCheckPassAll) Check(value []byte) error {
	return nil
}