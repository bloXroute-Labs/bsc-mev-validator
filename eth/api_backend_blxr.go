package eth

import (
	"context"

	"github.com/ethereum/go-ethereum/internal/ethapi"
)

func (b *EthAPIBackend) RegisterValidator(ctx context.Context, args *ethapi.RegisterValidatorArgs) error {
	return b.eth.sentryProxy.RegisterValidator(ctx, args)
}

func (b *EthAPIBackend) ProposedBlock(ctx context.Context, args *ethapi.ProposedBlockArgs) (any, error) {
	return b.eth.sentryProxy.ProposedBlock(ctx, args)
}
