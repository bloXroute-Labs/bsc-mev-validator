package ethapi

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func (b testBackend) ProposedBlock(context.Context, string, *big.Int, common.Hash, *big.Int, uint64, uint64, types.Transactions, map[common.Hash]struct{}) (time.Duration, error) {
	return 0, nil
}

func (b testBackend) AddRelay(context.Context, string) error    { return nil }
func (b testBackend) RemoveRelay(context.Context, string) error { return nil }
