package les

import (
	"context"

	"github.com/ethereum/go-ethereum/internal/ethapi"
)

func (b *LesApiBackend) RegisterValidator(context.Context, *ethapi.RegisterValidatorArgs) error {
	return nil
}

func (b *LesApiBackend) ProposedBlock(context.Context, *ethapi.ProposedBlockArgs) (any, error) {
	return nil, nil
}
