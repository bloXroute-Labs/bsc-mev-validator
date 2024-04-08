package eth

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/bloXroute-Labs/bx-mev-tools/pkg/ctxutil"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	sentryLogPrefix = "[SENTRY] "

	methodRegisterValidator = "mev_registerValidator"
	methodProposedBlock     = "mev_proposedBlock"
	methodBlockNumber       = "mev_blockNumber"
)

type (
	Forwarder interface {
		Forward(ctx context.Context, result interface{}, method string, args ...json.RawMessage) error
	}

	ContextCaller interface {
		CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
	}

	SentryProxyOpt = func(proxy *SentryProxy)

	SentryProxy struct {
		miner  Forwarder
		relays []ContextCaller
	}
)

func NewSentryProxy(cfg *ethconfig.Config) *SentryProxy {
	proxy := &SentryProxy{}

	for _, uri := range cfg.SentryRelaysUri {
		client, err := rpc.Dial(uri)
		if err != nil {
			logSentryError("Failed to dial relay", "relayUri", uri, "err", err)
			continue
		}

		proxy.relays = append(proxy.relays, client)
	}

	if cfg.SentryMinerUri == "" {
		logSentryError("Miner URI is empty")
		return proxy
	}

	var err error
	proxy.miner, err = rpc.Dial(cfg.SentryMinerUri)
	if err != nil {
		logSentryError("Failed to dial miner", "minerUri", cfg.SentryMinerUri, "err", err)
	}

	return proxy
}

// RegisterValidator register a validator
func (s *SentryProxy) RegisterValidator(ctx context.Context, args ethapi.RegisterValidatorArgs) error {
	if len(s.relays) == 0 {
		s.logProxyingError(
			"No relay clients",
			"RegisterValidator", args, nil, nil)
		return nil
	}

	// set the flag to indicate that this is a sentry call
	args.IsSentry = true

	var (
		allErrors error
		result    any
	)

	for _, relayClient := range s.relays {
		if err := ctxutil.Valid(ctx); err != nil {
			allErrors = errors.Join(allErrors, err)
			return allErrors
		}

		if err := relayClient.CallContext(ctx, &result, methodRegisterValidator, args); err != nil {
			s.logProxyingError(
				"Failed to register validator",
				"RegisterValidator", args, result, err)
			allErrors = errors.Join(allErrors, err)
		}
	}

	return allErrors
}

// ProposedBlock add the block to the list of works
func (s *SentryProxy) ProposedBlock(ctx context.Context, block json.RawMessage) (any, error) {
	var result any

	if s.miner == nil {
		s.logProxyingError(
			"ProposedBlock: No miner client",
			"ProposedBlock", truncatePayload(block), nil, nil)
		return nil, nil
	}

	if err := s.miner.Forward(ctx, &result, methodProposedBlock, block); err != nil {
		s.logProxyingError(
			"ProposedBlock: Failed to propose block to validator",
			"ProposedBlock", truncatePayload(block), result, err)
		return nil, err
	}

	return result, nil
}

func (s *SentryProxy) BlockNumber(ctx context.Context) (hexutil.Uint64, error) {
	var result hexutil.Uint64

	if s.miner == nil {
		s.logProxyingError(
			"No miner client",
			"BlockNumber", nil, nil, nil)
		return 0, nil
	}

	if err := s.miner.Forward(ctx, &result, methodBlockNumber); err != nil {
		s.logProxyingError(
			"Failed to propose block to validator",
			"BlockNumber", nil, result, err)
		return 0, err
	}

	return result, nil
}

func (s *SentryProxy) logProxyingError(logName string, method string, args any, result any, err error) {
	logCtx := make([]any, 0)

	if err != nil {
		logCtx = append(logCtx, "err", err)
	}

	if args != nil {
		argsBytes, argsMarshalErr := json.Marshal(args)
		if argsMarshalErr != nil {
			logCtx = append(logCtx, "argsMarshalErr", argsMarshalErr)
		} else {
			logCtx = append(logCtx, "args", string(argsBytes))
		}
	}

	if result != nil {
		resultBytes, resultMarshalErr := json.Marshal(result)
		if resultMarshalErr != nil {
			logCtx = append(logCtx, "resultMarshalErr", resultMarshalErr)
		} else {
			logCtx = append(logCtx, "result", string(resultBytes))
		}
	}

	logCtx = append(logCtx, "method", method)

	logSentryError(logName, logCtx...)
}

func logSentryError(msg string, logCtx ...any) {
	log.Error(sentryLogPrefix+msg, logCtx...)
}

func truncatePayload(args json.RawMessage) ethapi.ProposedBlockArgs {
	result := ethapi.ProposedBlockArgs{}
	_ = json.Unmarshal(args, &result)

	result.Payload = nil
	return result
}
