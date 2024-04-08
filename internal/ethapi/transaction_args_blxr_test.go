package ethapi

import (
	"context"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func (b *backendMock) RegisterValidator(context.Context, RegisterValidatorArgs) error { return nil }

func (b *backendMock) ProposedBlock(context.Context, json.RawMessage) (any, error) {
	return nil, nil
}

func (b *backendMock) BlockNumber(context.Context) (hexutil.Uint64, error) {
	return 0, nil
}
