package ethapi

import (
	"context"

	"github.com/ethereum/go-ethereum/rpc"
)

type MEVBackend interface {
	RegisterValidator(ctx context.Context, args *RegisterValidatorArgs) error
	ProposedBlock(ctx context.Context, args *ProposedBlockArgs) (any, error)
}

func getMEVAPIs(apiBackend MEVBackend) []rpc.API {
	mevAPI := NewPublicMEVAPI(apiBackend)
	return []rpc.API{
		{Namespace: "mev", Service: mevAPI},
		{Namespace: "eth", Service: mevAPI},
	}
}
