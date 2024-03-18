package ethapi

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func (b *backendMock) ProposedBlock(context.Context, string, *big.Int, common.Hash, *big.Int, uint64, uint64, types.Transactions, map[common.Hash]struct{}) (time.Duration, error) {
	return 0, nil
}

func (b *backendMock) AddRelay(context.Context, string) error    { return nil }
func (b *backendMock) RemoveRelay(context.Context, string) error { return nil }
func (b *backendMock) BlockNumber(_ context.Context) uint64      { return 0 }
