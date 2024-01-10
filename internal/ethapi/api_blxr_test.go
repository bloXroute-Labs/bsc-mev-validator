package ethapi

import (
	"context"
)

func (b *testBackend) RegisterValidator(context.Context, *RegisterValidatorArgs) error { return nil }

func (b *testBackend) ProposedBlock(context.Context, *ProposedBlockArgs) (any, error) {
	return nil, nil
}
