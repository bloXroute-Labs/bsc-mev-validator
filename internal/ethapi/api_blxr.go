package ethapi

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

const timestampFormat = "2006-01-02 15:04:05.000000"

// RegisterValidatorArgs has the epoch and the uri to send the proposedBlock to the validator
type RegisterValidatorArgs struct {
	Data       hexutil.Bytes `json:"data"` // bytes of string with callback ProposedBlockUri
	Signature  hexutil.Bytes `json:"signature"`
	Namespace  string        `json:"namespace"`
	CommitHash string        `json:"commitHash"`
	GasCeil    uint64        `json:"gasCeil"`
}

// MevAPI provides an API to MEV endpoints.
type MevAPI struct{ b Backend }

// NewMevAPI creates a new MEV protocol API.
func NewMevAPI(b Backend) *MevAPI { return &MevAPI{b} }

type ProposedBlockArgs struct {
	MEVRelay         string          `json:"mevRelay,omitempty"`
	BlockNumber      rpc.BlockNumber `json:"blockNumber"`
	PrevBlockHash    common.Hash     `json:"prevBlockHash"`
	BlockReward      *big.Int        `json:"blockReward"`
	GasLimit         uint64          `json:"gasLimit"`
	GasUsed          uint64          `json:"gasUsed"`
	Payload          []hexutil.Bytes `json:"payload"`
	UnRevertedHashes []common.Hash   `json:"unRevertedHashes,omitempty"`
}

type ProposedBlockResponse struct {
	ReceivedAt        string        `json:"receivedAt"`
	SimulatedDuration time.Duration `json:"simulatedDuration"`
	ResponseSentAt    string        `json:"responseSentAt"`
}

// ProposedBlock will submit the block to the miner worker
func (s *MevAPI) ProposedBlock(ctx context.Context, args ProposedBlockArgs) (*ProposedBlockResponse, error) {
	progress := s.b.SyncProgress()
	if progress.CurrentBlock < progress.HighestBlock {
		return nil, fmt.Errorf(
			"syncing in the process. Current block: %d, highest block: %d",
			progress.CurrentBlock, progress.HighestBlock)
	}

	var (
		receivedAt = time.Now().UTC()
		txs        types.Transactions
	)

	if len(args.Payload) == 0 {
		return nil, errors.New("block missing txs")
	}

	if args.BlockNumber == 0 {
		return nil, errors.New("block missing blockNumber")
	}

	currentBlock := s.b.CurrentBlock()
	blockOnChain := currentBlock.Number
	proposedBlockNumber := big.NewInt(args.BlockNumber.Int64())

	if proposedBlockNumber.Cmp(blockOnChain) < 1 {
		log.Info(log.MEVPrefix+"Validating ProposedBlock failed", "blockNumber", args.BlockNumber.String(), "onChainBlockNumber", blockOnChain.String(), "onChainBlockHash", currentBlock.Hash().String(), "prevBlockHash", args.PrevBlockHash.String(), "mevRelay", args.MEVRelay)
		return nil, fmt.Errorf("blockNumber is incorrect. proposedBlockNumber: %v onChainBlockNumber: %v onChainBlockHash %v", args.BlockNumber, blockOnChain, currentBlock.Hash().String())
	}

	for _, encodedTx := range args.Payload {
		tx := new(types.Transaction)
		if err := tx.UnmarshalBinary(encodedTx); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}

	unRevertedHashes := make(map[common.Hash]struct{}, len(args.UnRevertedHashes))
	for _, hash := range args.UnRevertedHashes {
		unRevertedHashes[hash] = struct{}{}
	}

	simDuration, err := s.b.ProposedBlock(ctx, args.MEVRelay, proposedBlockNumber, args.PrevBlockHash, args.BlockReward, args.GasLimit, args.GasUsed, txs, unRevertedHashes)
	if err != nil {
		return nil, err
	}

	return &ProposedBlockResponse{
		ReceivedAt:        receivedAt.Format(timestampFormat),
		SimulatedDuration: simDuration,
		ResponseSentAt:    time.Now().UTC().Format(timestampFormat),
	}, nil
}

type AddRelayArgs struct {
	MEVRelay string `json:"mevRelay"`
}

// AddRelay will submit the block to the miner worker
func (s *MevAPI) AddRelay(ctx context.Context, args AddRelayArgs) error {
	return s.b.AddRelay(ctx, args.MEVRelay)
}

type RemoveRelayArgs struct {
	MEVRelay string `json:"mevRelay"`
}

// RemoveRelay will submit the block to the miner worker
func (s *MevAPI) RemoveRelay(ctx context.Context, args RemoveRelayArgs) error {
	return s.b.RemoveRelay(ctx, args.MEVRelay)
}

// BlockNumber returns the block number of the chain head.
func (s *MevAPI) BlockNumber(ctx context.Context) (hexutil.Uint64, error) {
	return hexutil.Uint64(s.b.BlockNumber(ctx)), nil
}
