package eth

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

func (b *EthAPIBackend) ProposedBlock(ctx context.Context, mevRelay string, blockNumber *big.Int, prevBlockHash common.Hash, reward *big.Int, gasLimit uint64, gasUsed uint64, txs types.Transactions, unRevertedHashes map[common.Hash]struct{}) (simDuration time.Duration, err error) {
	return b.eth.miner.ProposedBlock(ctx, mevRelay, blockNumber, prevBlockHash, reward, gasLimit, gasUsed, txs, unRevertedHashes)
}

func (b *EthAPIBackend) AddRelay(ctx context.Context, mevRelay string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return b.eth.miner.AddRelay(ctx, mevRelay)
	}
}

func (b *EthAPIBackend) RemoveRelay(ctx context.Context, mevRelay string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return b.eth.miner.RemoveRelay(mevRelay)
	}
}

func (b *EthAPIBackend) BlockNumber(ctx context.Context) uint64 {
	header, _ := b.HeaderByNumber(ctx, rpc.LatestBlockNumber) // latest header should always be available
	return header.Number.Uint64()
}
