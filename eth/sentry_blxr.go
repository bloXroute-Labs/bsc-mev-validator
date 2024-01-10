package eth

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/bloXroute-Labs/bx-mev-tools/pkg/ctxutil"

	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	sentryLogPrefix = "[SENTRY] "

	methodRegisterValidator = "mev_registerValidator"
	methodProposedBlock     = "mev_proposedBlock"
)

type sentryProxy struct {
	minerClient  *rpc.Client
	relayClients []*rpc.Client
}

func newSentryProxy(cfg *ethconfig.Config) *sentryProxy {
	return new(sentryProxy).setMinerUri(cfg).setRelayClients(cfg)
}

// RegisterValidator register a validator
func (s *sentryProxy) RegisterValidator(ctx context.Context, args *ethapi.RegisterValidatorArgs) (err error) {
	if len(s.relayClients) == 0 {
		s.logProxyingError("No relay clients", args, nil, nil)

		return
	}

	// set the flag to indicate that this is a sentry call
	args.IsSentry = true

	var (
		clientErr error
		result    any
	)

	for _, relayClient := range s.relayClients {
		if clientErr = ctxutil.Valid(ctx); clientErr != nil {
			err = errors.Join(err, clientErr)

			return
		}

		if clientErr = relayClient.CallContext(ctx, &result, methodRegisterValidator, args); clientErr == nil {
			continue
		}

		err = errors.Join(err, clientErr)

		s.logProxyingError("Failed to register validator", args, result, clientErr)
	}

	return
}

// ProposedBlock add the block to the list of works
func (s *sentryProxy) ProposedBlock(ctx context.Context, args *ethapi.ProposedBlockArgs) (result any, err error) {
	noPayloadArgs := *args
	noPayloadArgs.Payload = nil

	if s.minerClient == nil {
		s.logProxyingError("No miner client", noPayloadArgs, result, err)

		return
	}

	if err = s.minerClient.CallContext(ctx, &result, methodProposedBlock, args); err == nil {
		return
	}

	s.logProxyingError("Failed to propose block to validator", noPayloadArgs, result, err)

	return
}

func (s *sentryProxy) setMinerUri(cfg *ethconfig.Config) *sentryProxy {
	if len(cfg.SentryMinerUri) == 0 {
		s.logError("Miner URI is empty")

		return s
	}

	var err error

	if s.minerClient, err = rpc.Dial(cfg.SentryMinerUri); err != nil {
		s.logError("Failed to dial miner", "minerUri", cfg.SentryMinerUri, "err", err)
	}

	return s
}

func (s *sentryProxy) setRelayClients(cfg *ethconfig.Config) *sentryProxy {
	var (
		relayClient *rpc.Client
		err         error
	)

	for _, relayUri := range cfg.SentryRelaysUri {
		// since relay must have High Availability, we won't try to dial it again if it failed
		relayClient, err = rpc.Dial(relayUri)
		if err != nil {
			s.logError("Failed to dial relay", "relayUri", relayUri, "err", err)

			continue
		}

		s.relayClients = append(s.relayClients, relayClient)
	}

	return s
}

func (s *sentryProxy) logProxyingError(logName string, args any, result any, err error) {
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

	s.logError(logName, logCtx...)
}

func (s *sentryProxy) logError(msg string, logCtx ...any) { log.Error(sentryLogPrefix+msg, logCtx...) }
