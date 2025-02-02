package abiparser

import (
	"crypto/rsa"
	"strings"

	"github.com/bloxapp/ssv/utils/rsaencryption"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// ValidatorAddedEvent struct represents event received by the smart contract
type ValidatorAddedEvent struct {
	PublicKey          []byte
	OwnerAddress       common.Address
	OperatorPublicKeys [][]byte
	SharesPublicKeys   [][]byte
	EncryptedKeys      [][]byte
}

// OperatorAddedEvent struct represents event received by the smart contract
type OperatorAddedEvent struct {
	Name         string
	OwnerAddress common.Address
	PublicKey    []byte
}

// V2Abi parsing events from v2 abi contract
type V2Abi struct {
}

// ParseOperatorAddedEvent parses an OperatorAddedEvent
func (v2 *V2Abi) ParseOperatorAddedEvent(
	logger *zap.Logger,
	operatorPubKey string,
	data []byte,
	topics []common.Hash,
	contractAbi abi.ABI,
) (*OperatorAddedEvent, bool, bool, error) {
	var operatorAddedEvent OperatorAddedEvent
	err := contractAbi.UnpackIntoInterface(&operatorAddedEvent, "OperatorAdded", data)
	if err != nil {
		return nil, false, true, errors.Wrap(err, "failed to unpack OperatorAdded event")
	}
	outAbi, err := getOutAbi()
	if err != nil {
		return nil, false, false, err
	}
	pubKey, err := readOperatorPubKey(operatorAddedEvent.PublicKey, outAbi)
	if err != nil {
		return nil, false, true, errors.Wrap(err, "failed to read OperatorPublicKey")
	}
	operatorAddedEvent.PublicKey = []byte(pubKey)

	if len(topics) > 1 {
		operatorAddedEvent.OwnerAddress = common.HexToAddress(topics[1].Hex())
	} else {
		logger.Error("operator event missing topics. no owner address provided.")
	}
	isOperatorEvent := strings.EqualFold(pubKey, operatorPubKey)
	return &operatorAddedEvent, isOperatorEvent, false, nil
}

// ParseValidatorAddedEvent parses ValidatorAddedEvent
func (v2 *V2Abi) ParseValidatorAddedEvent(
	logger *zap.Logger,
	operatorPrivateKey *rsa.PrivateKey,
	data []byte,
	contractAbi abi.ABI,
) (*ValidatorAddedEvent, bool, bool, error) {
	var validatorAddedEvent ValidatorAddedEvent
	err := contractAbi.UnpackIntoInterface(&validatorAddedEvent, "ValidatorAdded", data)
	if err != nil {
		return nil, false, true, errors.Wrap(err, "Failed to unpack ValidatorAdded event")
	}

	var isOperatorEvent bool
	for i, operatorPublicKey := range validatorAddedEvent.OperatorPublicKeys {
		outAbi, err := getOutAbi()
		if err != nil {
			return nil, false, false, errors.Wrap(err, "failed to define ABI")
		}
		operatorPublicKey, err := readOperatorPubKey(operatorPublicKey, outAbi)
		if err != nil {
			return nil, false, true, errors.Wrap(err, "failed to read OperatorPublicKey")
		}

		validatorAddedEvent.OperatorPublicKeys[i] = []byte(operatorPublicKey) // set for further use in code
		if operatorPrivateKey == nil {
			continue
		}
		nodeOperatorPubKey, err := rsaencryption.ExtractPublicKey(operatorPrivateKey)
		if err != nil {
			return nil, false, false, errors.Wrap(err, "failed to extract public key")
		}
		if strings.EqualFold(operatorPublicKey, nodeOperatorPubKey) {
			out, err := outAbi.Unpack("method", validatorAddedEvent.EncryptedKeys[i])
			if err != nil {
				return nil, false, true, errors.Wrap(err, "failed to unpack EncryptedKey")
			}

			if encryptedSharePrivateKey, ok := out[0].(string); ok {
				decryptedSharePrivateKey, err := rsaencryption.DecodeKey(operatorPrivateKey, encryptedSharePrivateKey)
				decryptedSharePrivateKey = strings.Replace(decryptedSharePrivateKey, "0x", "", 1)
				if err != nil {
					return nil, false, false, errors.Wrap(err, "failed to decrypt share private key")
				}
				validatorAddedEvent.EncryptedKeys[i] = []byte(decryptedSharePrivateKey)
				isOperatorEvent = true
			}
		}
	}

	return &validatorAddedEvent, isOperatorEvent, false, nil
}
