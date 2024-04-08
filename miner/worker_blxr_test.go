package miner

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/assert"
)

const (
	testPreferableRelay    = "preferable"
	testNonPreferableRelay = "non-preferable"
)

var (
	testConfigWithPreferRelay = &Config{
		Recommit:        time.Second,
		GasCeil:         params.GenesisGasLimit,
		PreferMEVRelays: NewAcceptRelayMap(testPreferableRelay),
	}
)

func Test_worker_isBlockMoreProfitable(t *testing.T) {
	t.Run("no blocks information", func(t *testing.T) {
		var (
			w                  = worker{}
			blockNum    uint64 = 200
			prevHash           = hash("HASH-199").String()
			blockReward        = big.NewInt(1000)
		)
		profitable, reward, work := w.isBlockMoreProfitableUnsafe(blockNum, prevHash, blockReward)

		assert.True(t, profitable)
		assert.Equal(t, uint64(0), reward.Uint64())
		assert.Nil(t, work)
	})

	t.Run("no info about the block", func(t *testing.T) {
		var (
			w = worker{
				bestProposedBlockInfo: blockInfoBuilder(199, hash("HASH-198"), 100, testPreferableRelay),
			}
			blockNum    uint64 = 200
			prevHash           = hash("HASH-199").String()
			blockReward        = big.NewInt(1000)
		)
		profitable, reward, work := w.isBlockMoreProfitableUnsafe(blockNum, prevHash, blockReward)

		assert.True(t, profitable)
		assert.Equal(t, uint64(0), reward.Uint64())
		assert.Nil(t, work)
	})

	t.Run("no info about the hash", func(t *testing.T) {
		var (
			w = worker{
				// imagine we know Block-200 on top of Block-198
				bestProposedBlockInfo: blockInfoBuilder(200, hash("HASH-198"), 100, testPreferableRelay),
			}
			blockNum    uint64 = 200
			prevHash           = hash("HASH-199").String()
			blockReward        = big.NewInt(1000)
		)
		profitable, reward, work := w.isBlockMoreProfitableUnsafe(blockNum, prevHash, blockReward)

		assert.True(t, profitable)
		assert.Equal(t, uint64(0), reward.Uint64())
		assert.Nil(t, work)
	})

	t.Run("more profitable", func(t *testing.T) {
		var (
			w = worker{
				bestProposedBlockInfo: blockInfoBuilder(200, hash("HASH-199"), 100, testPreferableRelay),
			}
			blockNum    uint64 = 200
			prevHash           = hash("HASH-199").String()
			blockReward        = big.NewInt(1000)
		)
		profitable, reward, work := w.isBlockMoreProfitableUnsafe(blockNum, prevHash, blockReward)

		assert.True(t, profitable)
		assert.Equal(t, uint64(100), reward.Uint64())
		assert.NotNil(t, work)
	})

	t.Run("less profitable", func(t *testing.T) {
		var (
			w = worker{
				bestProposedBlockInfo: blockInfoBuilder(200, hash("HASH-199"), 5000, testPreferableRelay),
			}
			blockNum    uint64 = 200
			prevHash           = hash("HASH-199").String()
			blockReward        = big.NewInt(1000)
		)
		profitable, reward, work := w.isBlockMoreProfitableUnsafe(blockNum, prevHash, blockReward)

		assert.False(t, profitable)
		assert.Equal(t, uint64(5000), reward.Uint64())
		assert.NotNil(t, work)
	})
}

func requireMutexIsNotBlocked(t *testing.T, w *worker, block ProposedBlock) {
	t.Run("test if previous call has not blocked mutex", func(t *testing.T) {
		var (
			done        = make(chan struct{})
			ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
		)
		defer cancel()

		go func() {
			w.handleProposedBlock(&block)
			close(done)
		}()

		select {
		case <-ctx.Done():
			t.Fatalf("prevous call didn't unlock mutex")
		case <-done:
			// fine
		}
	})
}

func Test_worker_handleProposedBlock(t *testing.T) {
	t.Run("no blocks information", func(t *testing.T) {
		var (
			w = worker{
				bestProposedBlockLock: sync.RWMutex{},
				bestProposedBlockInfo: make(map[uint64]bestProposedWorks),
				config:                testConfigWithPreferRelay,
			}
			block = proposedBlockBuilder(200, hash("HASH-199"), 1000, testPreferableRelay)
		)

		accepted, bestReward := w.handleProposedBlock(&block)
		assert.True(t, accepted)
		assert.Equal(t, zeroReward, bestReward)

		acceptedBlock := w.bestProposedBlockInfo[200][block.args.prevBlockHash.String()]
		assert.Equal(t, block.simulatedWork, acceptedBlock)

		requireMutexIsNotBlocked(t, &w, block)
	})

	t.Run("no current block information", func(t *testing.T) {
		var (
			w = worker{
				bestProposedBlockLock: sync.RWMutex{},
				bestProposedBlockInfo: blockInfoBuilder(199, hash("HASH-198"), 1000, testPreferableRelay),
				config:                testConfigWithPreferRelay,
			}
			block = proposedBlockBuilder(200, hash("HASH-199"), 1000, testPreferableRelay)
		)

		accepted, bestReward := w.handleProposedBlock(&block)
		assert.True(t, accepted)
		assert.Equal(t, zeroReward, bestReward)

		acceptedBlock := w.bestProposedBlockInfo[200][block.args.prevBlockHash.String()]
		assert.Equal(t, block.simulatedWork, acceptedBlock)

		requireMutexIsNotBlocked(t, &w, block)
	})

	t.Run("no blockNumber+prevBlockHash information", func(t *testing.T) {
		var (
			w = worker{
				bestProposedBlockLock: sync.RWMutex{},
				bestProposedBlockInfo: blockInfoBuilder(200, hash("HASH-198"), 1000, testPreferableRelay),
				config:                testConfigWithPreferRelay,
			}
			block = proposedBlockBuilder(200, hash("HASH-199"), 1000, testPreferableRelay)
		)

		accepted, bestReward := w.handleProposedBlock(&block)
		assert.True(t, accepted)
		assert.Equal(t, zeroReward, bestReward)

		acceptedBlock := w.bestProposedBlockInfo[200][block.args.prevBlockHash.String()]
		assert.Equal(t, block.simulatedWork, acceptedBlock)

		requireMutexIsNotBlocked(t, &w, block)
	})

	t.Run("more profitable proposal", func(t *testing.T) {
		var (
			w = worker{
				bestProposedBlockLock: sync.RWMutex{},
				bestProposedBlockInfo: blockInfoBuilder(200, hash("HASH-199"), 500, testPreferableRelay),
				config:                testConfigWithPreferRelay,
			}
			block = proposedBlockBuilder(200, hash("HASH-199"), 1000, testPreferableRelay)
		)

		accepted, bestReward := w.handleProposedBlock(&block)
		assert.True(t, accepted)
		assert.Equal(t, int64(500), bestReward.Int64())

		acceptedBlock := w.bestProposedBlockInfo[200][block.args.prevBlockHash.String()]
		assert.Equal(t, block.simulatedWork, acceptedBlock)

		requireMutexIsNotBlocked(t, &w, block)
	})

	t.Run("less profitable proposal", func(t *testing.T) {
		var (
			w = worker{
				bestProposedBlockLock: sync.RWMutex{},
				bestProposedBlockInfo: blockInfoBuilder(200, hash("HASH-199"), 5000, testPreferableRelay),
				config:                testConfigWithPreferRelay,
			}
			block = proposedBlockBuilder(200, hash("HASH-199"), 1000, testPreferableRelay)
		)

		accepted, bestReward := w.handleProposedBlock(&block)
		assert.False(t, accepted)
		assert.Equal(t, int64(5000), bestReward.Int64())

		// do not update block info
		acceptedBlock := w.bestProposedBlockInfo[200][block.args.prevBlockHash.String()]
		assert.NotEqual(t, block.simulatedWork, acceptedBlock)

		requireMutexIsNotBlocked(t, &w, block)
	})

	t.Run("prefer bloXroute's blocks", func(t *testing.T) {
		var (
			w = worker{
				bestProposedBlockLock: sync.RWMutex{},
				config:                testConfigWithPreferRelay,
				bestProposedBlockInfo: blockInfoBuilder(200, hash("HASH-199"), 50, testPreferableRelay),
			}
			block1 = proposedBlockBuilder(200, hash("HASH-199"), 300, testPreferableRelay)
			block2 = proposedBlockBuilder(200, hash("HASH-199"), 500, testNonPreferableRelay)
			block3 = proposedBlockBuilder(200, hash("HASH-199"), 100, testPreferableRelay)
		)

		accepted, bestReward := w.handleProposedBlock(&block1)
		assert.True(t, accepted)
		assert.Equal(t, int64(50), bestReward.Int64())

		requireMutexIsNotBlocked(t, &w, block1)

		accepted, bestReward = w.handleProposedBlock(&block2)
		assert.False(t, accepted)
		assert.Equal(t, block2.args.blockReward.Int64(), bestReward.Int64())

		requireMutexIsNotBlocked(t, &w, block2)

		accepted, bestReward = w.handleProposedBlock(&block3)
		assert.False(t, accepted)
		assert.Equal(t, block1.args.blockReward.Int64(), bestReward.Int64())

		requireMutexIsNotBlocked(t, &w, block3)
	})

	t.Run("no current block information - prefer bloXroute's blocks", func(t *testing.T) {
		var (
			w = worker{
				bestProposedBlockLock: sync.RWMutex{},
				bestProposedBlockInfo: blockInfoBuilder(199, hash("HASH-198"), 1000, testPreferableRelay),
				config:                testConfigWithPreferRelay,
			}
			block1 = proposedBlockBuilder(200, hash("HASH-199"), 1000, testNonPreferableRelay)
			block2 = proposedBlockBuilder(200, hash("HASH-199"), 1000, testPreferableRelay)
		)

		accepted, bestReward := w.handleProposedBlock(&block1)
		assert.False(t, accepted)
		assert.Equal(t, block1.args.blockReward.Int64(), bestReward.Int64())

		acceptedBlock := w.bestProposedBlockInfo[200][block1.args.prevBlockHash.String()]
		assert.Nil(t, acceptedBlock)

		requireMutexIsNotBlocked(t, &w, block1)

		accepted, bestReward = w.handleProposedBlock(&block2)
		assert.True(t, accepted)
		assert.Equal(t, zeroReward, bestReward)

		acceptedBlock = w.bestProposedBlockInfo[200][block2.args.prevBlockHash.String()]
		assert.Equal(t, block2.simulatedWork, acceptedBlock)

		requireMutexIsNotBlocked(t, &w, block2)
	})
}

func blockInfoBuilder(num uint64, hash common.Hash, reward int64, mevRelay string) map[uint64]bestProposedWorks {
	return map[uint64]bestProposedWorks{
		num: map[string]*bestProposedWork{
			hash.String(): {
				work:     &environment{},
				reward:   big.NewInt(reward),
				mevRelay: mevRelay,
			},
		},
	}
}

func hash(v string) common.Hash {
	return common.BytesToHash([]byte(v))
}

func proposedBlockBuilder(num int64, hash common.Hash, reward int64, mevRelay string) ProposedBlock {
	args := ProposedBlockArgs{
		blockNumber:   big.NewInt(num),
		prevBlockHash: hash,
		blockReward:   big.NewInt(reward),
		mevRelay:      mevRelay,
	}

	work := bestProposedWork{
		work:   &environment{},
		reward: big.NewInt(reward),
	}

	simDuration := time.Millisecond * 100

	return ProposedBlock{
		args:          &args,
		simulatedWork: &work,
		simDuration:   simDuration,
	}
}
