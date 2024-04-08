package ethapi

import (
	"context"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func (b *testBackend) RegisterValidator(context.Context, RegisterValidatorArgs) error { return nil }

func (b *testBackend) ProposedBlock(context.Context, json.RawMessage) (any, error) {
	return nil, nil
}

func (b *testBackend) BlockNumber(context.Context) (hexutil.Uint64, error) {
	return 0, nil
}
