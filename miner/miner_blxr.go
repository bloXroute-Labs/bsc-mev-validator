package miner

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bloXroute-Labs/bx-mev-tools/pkg/maputil"
	"github.com/ethereum/go-ethereum/blxr/blxerr"
	"github.com/ethereum/go-ethereum/blxr/version"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

type ClientMap map[string]*rpc.Client

type ClientMapping struct {
	mx        *sync.RWMutex
	clientMap ClientMap
}

func NewClientMap(relays []string) *ClientMapping {
	c := &ClientMapping{
		mx:        new(sync.RWMutex),
		clientMap: make(ClientMap),
	}

	for _, relay := range relays {
		client, err := rpc.Dial(relay)
		if err != nil {
			log.Warn(log.MEVPrefix+"Failed to dial MEV relay", "dest", relay, "err", err)
			continue
		}

		c.clientMap[relay] = client
	}

	return c
}

func (c *ClientMapping) Len() int {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return len(c.clientMap)
}

func (c *ClientMapping) Mapping() ClientMap {
	clientMap := make(ClientMap, len(c.clientMap))

	c.mx.RLock()
	for k, v := range c.clientMap {
		clientMap[k] = v
	}
	c.mx.RUnlock()

	return clientMap
}

func (c *ClientMapping) Get(relay string) (*rpc.Client, bool) {
	c.mx.RLock()
	client, ok := c.clientMap[relay]
	c.mx.RUnlock()

	return client, ok
}

func (c *ClientMapping) Add(relay string) (*rpc.Client, error) {
	c.mx.Lock()
	defer c.mx.Unlock()

	client, err := rpc.Dial(relay)
	if err != nil {
		return nil, err
	}

	c.clientMap[relay] = client

	return client, nil
}

func (c *ClientMapping) Remove(relay string) error {
	c.mx.Lock()
	defer c.mx.Unlock()

	if _, ok := c.clientMap[relay]; !ok {
		return fmt.Errorf("relay %s not found", relay)
	}

	delete(c.clientMap, relay)

	return nil
}

type ProposedBlockResult struct {
	SimDuration time.Duration
	Accepted    bool
	BestReward  *big.Int
}

// ProposedBlock add the block to the list of works.
// In case if the block cannot be accepted due to it is less profitable than the validator has,
// it returns blxerr.ProposedBlockLessProfitableErr error with a minimal reward threshold inside.
func (miner *Miner) ProposedBlock(ctx context.Context, mevRelay string, blockNumber *big.Int, prevBlockHash common.Hash, reward *big.Int, gasLimit uint64, gasUsed uint64, txs types.Transactions, unReverted map[common.Hash]struct{}) (simDuration time.Duration, err error) {
	var (
		isBlockSkipped bool
		simWork        *bestProposedWork
	)

	endOfProposingWindow := time.Unix(int64(miner.eth.BlockChain().CurrentBlock().Time+miner.worker.chainConfig.Parlia.Period), 0).Add(-miner.worker.config.DelayLeftOver)

	timeout := time.Until(endOfProposingWindow)
	if timeout <= 0 {
		err = fmt.Errorf("proposed block is too late, end of proposing window %s, appeared %s later", endOfProposingWindow, common.PrettyDuration(timeout))
		return
	}

	proposingCtx, proposingCancel := context.WithTimeout(ctx, timeout)
	defer proposingCancel()

	currentGasLimit := atomic.LoadUint64(miner.worker.currentGasLimit)
	previousBlockGasLimit := atomic.LoadUint64(miner.worker.prevBlockGasLimit)
	defer func() {
		logCtx := []any{
			"blockNumber", blockNumber,
			"mevRelay", mevRelay,
			"prevBlockHash", prevBlockHash.Hex(),
			"proposedReward", reward,
			"gasLimit", gasLimit,
			"gasUsed", gasUsed,
			"txCount", len(txs),
			"unRevertedCount", len(unReverted),
			"isBlockSkipped", isBlockSkipped,
			"currentGasLimit", currentGasLimit,
			"timestamp", time.Now().UTC().Format(timestampFormat),
			"simDuration", simDuration,
		}

		if err != nil {
			logCtx = append(logCtx, "err", err)
		}

		log.Debug(log.MEVPrefix+"Received proposedBlock", logCtx...)
	}()
	isBlockSkipped = gasUsed > currentGasLimit
	if isBlockSkipped {
		err = fmt.Errorf("proposed block gasUsed %v exceeds the current block gas limit %v", gasUsed, currentGasLimit)
		return
	}
	desiredGasLimit := core.CalcGasLimit(previousBlockGasLimit, miner.worker.config.GasCeil)
	if desiredGasLimit != gasLimit {
		log.Warn(log.MEVPrefix+"proposedBlock has wrong gasLimit", "MEVRelay", mevRelay, "blockNumber", blockNumber, "validatorGasLimit", desiredGasLimit, "proposedBlockGasLimit", gasLimit)
		err = fmt.Errorf("proposed block gasLimit %v is different than the validator gasLimit %v", gasLimit, desiredGasLimit)
		return
	}
	args := &ProposedBlockArgs{
		mevRelay:      mevRelay,
		blockNumber:   blockNumber,
		prevBlockHash: prevBlockHash,
		blockReward:   reward,
		gasLimit:      gasLimit,
		gasUsed:       gasUsed,
		txs:           txs,
		unReverted:    unReverted,
	}
	simWork, simDuration, err = miner.worker.simulateProposedBlock(proposingCtx, args)
	if err != nil {
		err = fmt.Errorf("processing and simulating proposedBlock failed, %v", err)
		return
	}
	if simWork == nil {
		//  do not return error, when the block is skipped
		return
	}

	select {
	case <-proposingCtx.Done():
		return simDuration, errors.WithMessage(proposingCtx.Err(), "failed to propose block due to context timeout")
	default:
		// unblock and continue
	}

	accepted, bestReward := miner.worker.handleProposedBlock(&ProposedBlock{args: args, simulatedWork: simWork, simDuration: simDuration})
	if !accepted {
		return simDuration, blxerr.ProposedBlockLessProfitableErr{RewardThreshold: bestReward}
	}

	return simDuration, nil
}

func (miner *Miner) registerValidator() {
	log.Info(log.MEVPrefix + "register validator to MEV relays")
	registerValidatorArgs := &ethapi.RegisterValidatorArgs{
		Data:       []byte(miner.proposedBlockUri),
		Signature:  miner.signedProposedBlockUri,
		Namespace:  miner.proposedBlockNamespace,
		CommitHash: version.CommitHash(),
		GasCeil:    miner.worker.config.GasCeil,
	}
	for dest, destClient := range miner.mevRelays.Mapping() {
		go func(dest string, destinationClient *rpc.Client, registerValidatorArgs *ethapi.RegisterValidatorArgs) {
			var result any

			if err := destinationClient.Call(
				&result, "eth_registerValidator", registerValidatorArgs,
			); err != nil {
				log.Warn(log.MEVPrefix+"Failed to register validator to MEV relay", "dest", dest, "err", err)
				return
			}

			log.Debug(log.MEVPrefix+"register validator to MEV relay", "dest", dest, "result", result)
		}(dest, destClient, registerValidatorArgs)
	}
}

func (miner *Miner) AddRelay(ctx context.Context, relay string) error {
	client, err := miner.mevRelays.Add(relay)
	if err != nil {
		return err
	}

	log.Info(log.MEVPrefix+"register validator to MEV relay", "dest", relay)
	registerValidatorArgs := &ethapi.RegisterValidatorArgs{
		Data:       []byte(miner.proposedBlockUri),
		Signature:  miner.signedProposedBlockUri,
		Namespace:  miner.proposedBlockNamespace,
		CommitHash: version.CommitHash(),
		GasCeil:    miner.worker.config.GasCeil,
	}

	var result any

	if err = client.CallContext(
		ctx, &result, "eth_registerValidator", registerValidatorArgs,
	); err != nil {
		log.Warn(log.MEVPrefix+"Failed to register validator to MEV relay", "dest", relay, "err", err)
		return err
	}

	log.Debug(log.MEVPrefix+"register validator to MEV relay", "dest", relay, "result", result)

	return nil
}

func (miner *Miner) RemoveRelay(relay string) error {
	return miner.mevRelays.Remove(relay)
}

func isNewEpoch(block *types.Block) bool {
	return block.NumberU64()%params.BSCChainConfig.Parlia.Epoch == 0
}

// AcceptRelayMap is a map of MEV relays that the miner accepts
type AcceptRelayMap struct {
	relays *maputil.Set[string]
	len    int
}

// NewAcceptRelayMap creates a new AcceptRelayMap
func NewAcceptRelayMap(relays ...string) *AcceptRelayMap {
	return new(AcceptRelayMap).fromSlice(relays...)
}

func (m *AcceptRelayMap) fromSlice(relays ...string) *AcceptRelayMap {
	m.relays = maputil.SetOf(relays...)
	m.len = m.relays.Size()

	return m
}

// Accept returns true if the miner accepts the MEV relay
func (m *AcceptRelayMap) Accept(mevRelay string) bool {
	if m.len == 0 || m.relays == nil {
		return true
	}

	return m.relays.Has(mevRelay)
}

// MarshalText converts the AcceptRelayMap object into a text representation.
func (m *AcceptRelayMap) MarshalText() (text []byte, err error) {
	acceptRelays := make([]string, 0, m.len)
	m.relays.Each(func(key string) bool { acceptRelays = append(acceptRelays, key); return true })

	return json.Marshal(acceptRelays)
}

// UnmarshalText parses the text representation back into an AcceptRelayMap object.
func (m *AcceptRelayMap) UnmarshalText(text []byte) (err error) {
	var acceptRelays []string
	if err = json.Unmarshal(text, &acceptRelays); err != nil {
		return
	}

	m.fromSlice(acceptRelays...)

	return
}
