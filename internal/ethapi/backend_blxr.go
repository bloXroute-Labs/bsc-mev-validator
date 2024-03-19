package ethapi

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

// MEVBackend interface provides the common API services (that are provided by
// both full and light clients) with access to MEV functions.
type MEVBackend interface {
	ProposedBlock(ctx context.Context, mevRelay string, blockNumber *big.Int, prevBlockHash common.Hash, reward *big.Int, gasLimit uint64, gasUsed uint64, txs types.Transactions, unRevertedHashes map[common.Hash]struct{}) (simDuration time.Duration, err error)
	AddRelay(ctx context.Context, mevRelay string) error
	RemoveRelay(ctx context.Context, mevRelay string) error
	BlockNumber(ctx context.Context) uint64
}

func getMEVAPIs(apiBackend Backend) []rpc.API {
	mevAPI := NewMevAPI(apiBackend)
	return []rpc.API{
		{Namespace: "mev", Service: mevAPI},
		{Namespace: "eth", Service: mevAPI},
	}
}
