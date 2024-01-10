package les

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func (b *LesApiBackend) ProposedBlock(context.Context, string, *big.Int, common.Hash, *big.Int, uint64, uint64, types.Transactions, map[common.Hash]struct{}) (simDuration time.Duration, err error) {
	return
}

func (b *LesApiBackend) AddRelay(context.Context, string) error {
	return nil
}

func (b *LesApiBackend) RemoveRelay(context.Context, string) error {
	return nil
}
