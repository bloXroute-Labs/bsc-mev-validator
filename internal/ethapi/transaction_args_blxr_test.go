package ethapi

import (
	"context"
)

func (b *backendMock) RegisterValidator(context.Context, *RegisterValidatorArgs) error { return nil }

func (b *backendMock) ProposedBlock(context.Context, *ProposedBlockArgs) (any, error) {
	return nil, nil
}
