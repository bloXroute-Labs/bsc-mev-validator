package miner

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

const (
	// inTurnDifficulty when validator has main proposer duty
	inTurnDifficulty = 2

	//callerType to getBestWork
	callerTypeGenerateWork = "generateWork"
	callerTypeCommitWork   = "commitWork"

	// timestamp format
	timestampFormat = "2006-01-02 15:04:05.000000"
)

var zeroReward = big.NewInt(0)

// ProposedBlockArgs defines the argument of a proposed block
type ProposedBlockArgs struct {
	mevRelay      string
	blockNumber   *big.Int
	prevBlockHash common.Hash
	blockReward   *big.Int
	gasLimit      uint64
	gasUsed       uint64
	txs           types.Transactions
	unReverted    map[common.Hash]struct{}
}

// ProposedBlock defines the argument of a proposed block and simulated work
type ProposedBlock struct {
	args          *ProposedBlockArgs
	simulatedWork *bestProposedWork
	simDuration   time.Duration
}

type bestProposedWorks map[string]*bestProposedWork

// discard is unsafe operation, caller needs to protect before performing discard
func (b bestProposedWorks) discard() {
	for prevHash, w := range b {
		if w.work != nil {
			w.work.discard()
		}
		delete(b, prevHash)
	}
}

type bestProposedWork struct {
	work     *environment
	reward   *big.Int
	mevRelay string
}

func (w *worker) AcceptProposedBlock(mevRelay string) bool {
	if w.config == nil || w.config.PreferMEVRelays == nil {
		return true
	}

	return w.config.PreferMEVRelays.Accept(mevRelay)
}

// isBlockMoreProfitableUnsafe is a concurrency unsafe implementation of isBlockMoreProfitable.
func (w *worker) isBlockMoreProfitableUnsafe(num uint64, prevHash string, reward *big.Int) (bool, *big.Int, *environment) {
	// NOTE: if you need to add more returning values, please create a structure to return to prevent growing values.

	bestWorks, ok := w.bestProposedBlockInfo[num]
	if !ok || bestWorks == nil {
		return true, big.NewInt(0), nil
	}

	best, ok := bestWorks[prevHash]
	if !ok || best == nil || best.reward == nil {
		return true, big.NewInt(0), nil
	}

	// compare if the best knowing reward is higher than the proposed block reward
	return reward.Cmp(best.reward) == 1, best.reward, best.work
}

// isBlockMoreProfitable is a concurrency safe function to return true if the given block is more profitable than knowing blocks.
// The function also returns reward and work of the best known block.
func (w *worker) isBlockMoreProfitable(num uint64, prevHash string, reward *big.Int) (bool, *big.Int, *environment) {
	w.bestProposedBlockLock.RLock()
	defer w.bestProposedBlockLock.RUnlock()

	return w.isBlockMoreProfitableUnsafe(num, prevHash, reward)
}

// return true if block was accepted and previous best reward
func (w *worker) handleProposedBlock(block *ProposedBlock) (bool, *big.Int) {
	var (
		blockNum      = block.args.blockNumber.Uint64()
		prevBlockHash = block.args.prevBlockHash.String()
		blockReward   = block.simulatedWork.reward
	)

	if !w.AcceptProposedBlock(block.args.mevRelay) {
		log.Debug(log.MEVPrefix+"Received proposedBlock from non-preferred mevRelay", "blockNumber", blockNum, "prevBlockHash", prevBlockHash, "newProposedBlockReward", blockReward, "newProposedBlockGasLimit", block.args.gasLimit, "newProposedBlockGasUsed", block.args.gasUsed, "newProposedBlockTxCount", len(block.args.txs), "mevRelay", block.args.mevRelay, "timestamp", time.Now().UTC().Format(timestampFormat))
		return false, blockReward
	}

	// Skip if the proposed block is less profitable than a known one.
	if profitable, bestReward, _ := w.isBlockMoreProfitable(blockNum, prevBlockHash, blockReward); !profitable {
		log.Debug(log.MEVPrefix+"Skipping proposedBlock", "blockNumber", blockNum, "prevBlockHash", prevBlockHash, "newProposedBlockReward", blockReward, "previousProposedBlockReward", bestReward, "newProposedBlockGasLimit", block.args.gasLimit, "newProposedBlockGasUsed", block.args.gasUsed, "newProposedBlockTxCount", len(block.args.txs), "mevRelay", block.args.mevRelay, "timestamp", time.Now().UTC().Format(timestampFormat))
		return false, bestReward
	}

	w.bestProposedBlockLock.Lock()
	defer w.bestProposedBlockLock.Unlock()

	bestWorks, ok := w.bestProposedBlockInfo[blockNum]
	// It's the first block for this block number
	if (!ok || bestWorks == nil) && block.simulatedWork.work != nil {
		w.bestProposedBlockInfo[blockNum] = bestProposedWorks{prevBlockHash: block.simulatedWork}
		log.Info(log.MEVPrefix+"Received proposedBlock, this is the first proposed block for this block number and previous block hash", "blockNumber", block.args.blockNumber, "prevBlockHash", block.args.prevBlockHash.String(), "newProposedBlockReward", blockReward, "newProposedBlockGasUsed", block.args.gasUsed, "mevRelay", block.args.mevRelay, "newProposedBlockTxCount", len(block.args.txs), "simulatedDuration", block.simDuration, "timestamp", time.Now().UTC().Format(timestampFormat))
		return true, zeroReward
	}

	// Double check if the proposed block is more profitable than a known block.
	// We need this check because of a possible map mutations between mutex RUnlock and mutex Lock.
	// bestReward and bestWork are Reward and Work of the best block we know excluding the proposed one.
	profitable, bestReward, bestWork := w.isBlockMoreProfitableUnsafe(blockNum, prevBlockHash, blockReward)
	if !profitable {
		log.Info(log.MEVPrefix+"Received proposedBlock reward is not higher than previously proposed block", "blockNumber", blockNum, "prevBlockHash", prevBlockHash, "previousProposedBlockReward", bestReward, "newProposedBlockReward", blockReward, "newProposedBlockTxCount", len(block.simulatedWork.work.txs), "newProposedBlockGasUsed", block.args.gasUsed, "mevRelay", block.args.mevRelay, "simulatedDuration", block.simDuration, "timestamp", time.Now().UTC().Format(timestampFormat))
		return false, bestReward
	}

	// The proposed block is more profitable than an existing one.
	// Override the best block with the current one.
	log.Info(log.MEVPrefix+"Received proposedBlock, replacing previously proposedBlock after simulation", "blockNumber", blockNum, "prevBlockHash", prevBlockHash, "previousProposedBlockReward", bestReward, "newProposedBlockReward", blockReward, "newProposedBlockTxCount", len(block.simulatedWork.work.txs), "mevRelay", block.args.mevRelay, "newProposedBlockGasUsed", block.args.gasUsed, "simulatedDuration", block.simDuration, "timestamp", time.Now().UTC().Format(timestampFormat))
	w.bestProposedBlockInfo[blockNum][prevBlockHash] = block.simulatedWork

	if bestWork != nil {
		bestWork.discard()
	}

	return true, bestReward
}

// proposedLoop is responsible for generating and submitting sealing work based on
// proposed blocks
func (w *worker) proposedLoop() {
	chainBlockCh := make(chan core.ChainHeadEvent, chainHeadChanSize)

	chainBlockSub := w.eth.BlockChain().SubscribeChainBlockEvent(chainBlockCh)

	defer w.wg.Done()
	defer chainBlockSub.Unsubscribe()
	defer func() {
		if w.current != nil {
			w.current.discard()
		}
	}()

	for {
		select {
		case block := <-chainBlockCh:
			// each block will have its own interruptCh to stop work with a reason
			atomic.StoreUint64(w.prevBlockGasLimit, block.Block.GasLimit())
			worksToDiscard := make([]bestProposedWorks, 0)
			w.bestProposedBlockLock.Lock()
			for blockNumber, works := range w.bestProposedBlockInfo {
				if blockNumber <= w.chain.CurrentBlock().Number.Uint64() {
					worksToDiscard = append(worksToDiscard, works)
					delete(w.bestProposedBlockInfo, blockNumber)
				}
			}
			w.bestProposedBlockLock.Unlock()
			for _, works := range worksToDiscard {
				works.discard()
			}

		// System stopped
		case <-w.exitCh:
			return
		case <-chainBlockSub.Err():
			return
		}
	}
}

func (w *worker) getBestWorkBetweenInternalAndProposedBlock(internalWork *environment, callerType string) *environment {
	var (
		preferProposedBlock          bool
		internalBlockReward          = new(big.Int).Set(internalWork.state.GetBalance(consensus.SystemAddress)) // added for logging
		bestReward                   = new(big.Int).Set(internalBlockReward)
		bestWork                     = internalWork
		proposedBlockReward          = new(big.Int)
		validatorHasMainProposerDuty = internalWork.header.Difficulty.Cmp(new(big.Int).SetInt64(inTurnDifficulty)) == 0
	)

	if !validatorHasMainProposerDuty {
		// To prevent bundle leakage
		return bestWork
	}

	w.bestProposedBlockLock.RLock()
	defer w.bestProposedBlockLock.RUnlock()

	logCtx := []any{
		"blockNumber", bestWork.header.Number,
		"prevBlockHash", bestWork.header.ParentHash.String(),
		"type", callerType,
		"timestamp", time.Now().UTC().Format(timestampFormat),
	}

	works, ok := w.bestProposedBlockInfo[internalWork.header.Number.Uint64()]
	if !ok || works == nil {
		log.Info(log.MEVPrefix+"Prefer internal or proposedBlock", append(logCtx, "internalBlockReward", internalBlockReward, "preferProposedBlock", preferProposedBlock)...)
		return bestWork
	}

	if proposedBlock, exist := works[internalWork.header.ParentHash.String()]; exist && proposedBlock != nil && proposedBlock.work != nil {
		preferProposedBlock = proposedBlock.reward != nil && proposedBlock.reward.Cmp(bestReward) > 0 && w.AcceptProposedBlock(proposedBlock.mevRelay)
		proposedBlockReward.Set(proposedBlock.reward)
		if preferProposedBlock {
			logCtx = append(logCtx, "mevRelay", proposedBlock.mevRelay)
			if bestWork != nil {
				bestWork.discard()
			}
			bestWork = proposedBlock.work
			bestReward.Set(proposedBlock.reward)
		}
	}

	log.Info(log.MEVPrefix+"Prefer internal or proposedBlock", append(logCtx, "internalBlockReward", internalBlockReward, "preferProposedBlock", preferProposedBlock, "proposedBlockReward", proposedBlockReward)...)
	return bestWork
}

// fillTransactionsProposedBlock retrieves the pending transactions from the txpool and fills them
// into the given sealing block. The transaction selection and ordering strategy can
// be customized with the plugin in the future.
func (w *worker) fillTransactionsProposedBlock(ctx context.Context, env *environment, block *ProposedBlockArgs) (error, *big.Int) {
	gasLimit := env.header.GasLimit
	if env.gasPool == nil {
		env.gasPool = new(core.GasPool).AddGas(gasLimit)
		if w.chain.Config().IsEuler(env.header.Number) {
			env.gasPool.SubGas(params.SystemTxsGas * 3)
		} else {
			env.gasPool.SubGas(params.SystemTxsGas)
		}
	}

	var coalescedLogs []*types.Log
	// initialize bloom processors
	processorCapacity := 100
	bloomProcessors := core.NewAsyncReceiptBloomGenerator(processorCapacity)

	var blockReward *big.Int

	signal := commitInterruptNone
	for i, tx := range block.txs {
		select {
		default:
		case <-ctx.Done():
			err := ctx.Err()
			log.Trace("Filling of transaction stopped", "canceledAt", fmt.Sprintf("%d/%d", i, len(block.txs)), "err", err)
			return err, nil
		}

		// If we don't have enough gas for any further transactions then we're done
		if env.gasPool.Gas() < params.TxGas {
			log.Trace("Not enough gas for further transactions", "have", env.gasPool, "want", params.TxGas)
			signal = commitInterruptOutOfGas
			break
		}

		if tx.Protected() && !w.chainConfig.IsEIP155(env.header.Number) {
			return errors.New("block payload is incorrect"), nil
		}

		txHash := tx.Hash()

		// Start executing the transaction
		env.state.SetTxContext(txHash, env.tcount)

		_, err := w.commitTransactionOld(env, tx, bloomProcessors)
		if err != nil {
			log.Error(log.MEVPrefix+"Failed to commit transaction on proposedBlock", "blockNumber", block.blockNumber.String(), "fromMevRelay", block.mevRelay, "proposedTxs", len(block.txs), "failedTx", txHash.String())
			return err, nil
		}

		if env.receipts[len(env.receipts)-1].Status == types.ReceiptStatusFailed {
			if _, ok := block.unReverted[txHash]; ok {
				log.Warn(log.MEVPrefix+"bundle reverted proposedBlock", "blockNumber", block.blockNumber, "fromMevRelay", block.mevRelay, "revertedTx", txHash.Hex())
				return errors.New("bundle reverted"), nil
			}
		}
	}

	tcount := len(env.txs)
	blockReward = env.state.GetBalance(consensus.SystemAddress)
	log.Debug(log.MEVPrefix+"Processing proposedBlock", "blockNumber", block.blockNumber.String(),
		"fromMevRelay", block.mevRelay,
		"proposedReward", block.blockReward, "actualReward", blockReward,
		"proposedGasUsed", block.gasUsed, "actualGasUsed", env.receipts[len(env.receipts)-1].CumulativeGasUsed,
		"proposedTxsCount", len(block.txs), "actualTxsCount", tcount)

	if tcount < len(block.txs) {
		return errors.New("block parameters mismatch"), nil
	}

	bloomProcessors.Close()
	if !w.isRunning() && len(coalescedLogs) > 0 {
		// We don't push the pendingLogsEvent while we are sealing. The reason is that
		// when we are sealing, the worker will regenerate a sealing block every 3 seconds.
		// In order to avoid pushing the repeated pendingLog, we disable the pending log pushing.

		// make a copy, the state caches the logs and these logs get "upgraded" from pending to mined
		// logs by filling in the block hash when the block was mined by the local miner. This can
		// cause a race condition if a log was "upgraded" before the PendingLogsEvent is processed.
		cpy := make([]*types.Log, len(coalescedLogs))
		for i, l := range coalescedLogs {
			cpy[i] = new(types.Log)
			*cpy[i] = *l
		}
		w.pendingLogsFeed.Send(cpy)
	}
	return signalToErr(signal), blockReward
}

// simulateProposedBlock generates a sealing block based on a proposed block.
func (w *worker) simulateProposedBlock(ctx context.Context, proposedBlock *ProposedBlockArgs) (*bestProposedWork, time.Duration, error) {
	var (
		start  = time.Now()
		reward = new(big.Int).Set(proposedBlock.blockReward)
	)

	w.bestProposedBlockLock.RLock()
	if bestWorks, ok := w.bestProposedBlockInfo[proposedBlock.blockNumber.Uint64()]; ok && bestWorks != nil {
		previousProposedBlockWork, exist := bestWorks[proposedBlock.prevBlockHash.String()]
		if exist && previousProposedBlockWork != nil && previousProposedBlockWork.reward != nil {
			if previousProposedBlockWork.reward.Cmp(reward) > 0 {
				log.Debug(log.MEVPrefix+"Skipping proposedBlock", "blockNumber", proposedBlock.blockNumber, "prevBlockHash", proposedBlock.prevBlockHash.String(), "newProposedBlockReward", reward, "previousProposedBlockReward", previousProposedBlockWork.reward, "newProposedBlockGasLimit", proposedBlock.gasLimit, "newProposedBlockGasUsed", proposedBlock.gasUsed, "newProposedBlockTxCount", len(proposedBlock.txs), "mevRelay", proposedBlock.mevRelay, "timestamp", time.Now().UTC().Format(timestampFormat))
				w.bestProposedBlockLock.RUnlock()
				return nil, 0, nil
			}
		}
	}
	w.bestProposedBlockLock.RUnlock()

	// Set the coinbase if the worker is running or it's required
	var coinbase common.Address
	if w.isRunning() {
		if w.coinbase == (common.Address{}) {
			return nil, time.Since(start), errors.New("refusing to mine without etherbase")
		}
		coinbase = w.coinbase // Use the preset address as the fee recipient
	}

	work, err := w.prepareWork(&generateParams{
		timestamp: uint64(time.Now().Unix()),
		coinbase:  coinbase,
	})
	if err != nil {
		return nil, time.Since(start), err
	}
	defer work.discard()

	select {
	default:
	case <-ctx.Done():
		return nil, time.Since(start), ctx.Err()
	}

	// Fill transactions from the proposed block
	err, blockReward := w.fillTransactionsProposedBlock(ctx, work, proposedBlock)
	if err != nil {
		return nil, time.Since(start), err
	}

	nextBlock := big.NewInt(0).Add(big.NewInt(1), w.eth.BlockChain().CurrentBlock().Number)

	if nextBlock.Cmp(proposedBlock.blockNumber) != 0 {
		// block was changed during validation, need to ignore this proposedBlock
		return nil, time.Since(start), errors.New("chain changed")
	}

	bestWork := &bestProposedWork{
		work:     work.copy(),
		reward:   new(big.Int).Set(blockReward),
		mevRelay: proposedBlock.mevRelay,
	}
	totalDuration := time.Since(start)
	log.Debug(log.MEVPrefix+"simulated proposedBlock", "blockNumber", proposedBlock.blockNumber, "blockReward", blockReward, "fromMevRelay", proposedBlock.mevRelay, "duration", totalDuration, "timestamp", time.Now().UTC().Format(timestampFormat))
	return bestWork, totalDuration, nil
}
