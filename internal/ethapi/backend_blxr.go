package ethapi

import (
	"context"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

type MEVBackend interface {
	RegisterValidator(ctx context.Context, args RegisterValidatorArgs) error
	ProposedBlock(ctx context.Context, args json.RawMessage) (any, error)
	BlockNumber(ctx context.Context) (hexutil.Uint64, error)
}

func getMEVAPIs(apiBackend MEVBackend) []rpc.API {
	mevAPI := NewPublicMEVAPI(apiBackend)
	return []rpc.API{
		{Namespace: "mev", Service: mevAPI},
		{Namespace: "eth", Service: mevAPI},
	}
}
