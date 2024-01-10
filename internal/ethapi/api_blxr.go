package ethapi

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

// MevAPI provides an API to MEV endpoints.
type MevAPI struct{ b MEVBackend }

// NewPublicMEVAPI creates a new MEV protocol API.
func NewPublicMEVAPI(b MEVBackend) *MevAPI { return &MevAPI{b: b} }

// RegisterValidatorArgs represents the arguments to register a validator.
type RegisterValidatorArgs struct {
	Data       hexutil.Bytes `json:"data"` // bytes of string with callback ProposedBlockUri
	Signature  hexutil.Bytes `json:"signature"`
	IsSentry   bool          `json:"isSentry"`
	Namespace  string        `json:"namespace"`
	CommitHash string        `json:"commitHash,omitempty"`
	GasCeil    uint64        `json:"gasCeil"`
}

// RegisterValidator registers a validator for the next epoch to the pool of proposing destinations.
func (s *MevAPI) RegisterValidator(ctx context.Context, args *RegisterValidatorArgs) error {
	return s.b.RegisterValidator(ctx, args)
}

// ProposedBlockArgs are the arguments for the ProposedBlock RPC
type ProposedBlockArgs struct {
	MEVRelay         string          `json:"mevRelay,omitempty"`
	BlockNumber      rpc.BlockNumber `json:"blockNumber"`
	PrevBlockHash    common.Hash     `json:"prevBlockHash"`
	BlockReward      *big.Int        `json:"blockReward"`
	GasLimit         uint64          `json:"gasLimit"`
	GasUsed          uint64          `json:"gasUsed"`
	Payload          any             `json:"payload"`
	UnRevertedHashes []common.Hash   `json:"unRevertedHashes,omitempty"`
}

// ProposedBlock will submit the block to the miner worker
func (s *MevAPI) ProposedBlock(ctx context.Context, args *ProposedBlockArgs) (any, error) {
	return s.b.ProposedBlock(ctx, args)
}
