package eth

import (
	"context"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
)

func (b *EthAPIBackend) RegisterValidator(ctx context.Context, args ethapi.RegisterValidatorArgs) error {
	return b.eth.sentryProxy.RegisterValidator(ctx, args)
}

func (b *EthAPIBackend) ProposedBlock(ctx context.Context, args json.RawMessage) (any, error) {
	return b.eth.sentryProxy.ProposedBlock(ctx, args)
}

func (b *EthAPIBackend) BlockNumber(ctx context.Context) (hexutil.Uint64, error) {
	if b.eth.sentryProxy.miner != nil {
		return b.eth.sentryProxy.BlockNumber(ctx)
	}

	// fallback to native implementation and returning the local blocknumber without proxying
	header, _ := b.HeaderByNumber(context.Background(), rpc.LatestBlockNumber) // latest header should always be available
	return hexutil.Uint64(header.Number.Uint64()), nil
}
