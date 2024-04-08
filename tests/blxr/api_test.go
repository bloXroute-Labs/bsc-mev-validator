package blxr

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMevAPI simulates a backend where Sentry forwards requests to.
//
// It's fine that methods accept json.RawMessage instead of a real arguments what our real nodes are expecting
// because rpc server will unmarshal a valid argument JSON into whatever we have in the method signature.
// If we test and match that Sentry sends what we expect then we can be sure that everything will work in live.
type MockMevAPI struct {
	sync.Mutex

	ProposedBlockCalledArg string
	RegisterValidatorArg   string
}

func (m *MockMevAPI) ProposedBlock(_ context.Context, block json.RawMessage) (any, error) {
	m.Lock()
	defer m.Unlock()

	m.ProposedBlockCalledArg = string(block)
	return nil, nil
}

func (m *MockMevAPI) RegisterValidator(_ context.Context, arg json.RawMessage) (any, error) {
	m.Lock()
	defer m.Unlock()

	m.RegisterValidatorArg = string(arg)
	return nil, nil
}

func (m *MockMevAPI) BlockNumber(_ context.Context) (hexutil.Uint64, error) {
	return 100500, nil
}

func TestBlockNumber(t *testing.T) {
	t.Run("mev namespace over rpc", func(t *testing.T) {
		// run RPC server that emulates a miner the Sentry is forwarding requests to
		api := new(MockMevAPI)
		uri, stop := newRpcServer("mev", api)
		defer stop()

		// run test Ethereum Node Backend
		backend, _ := newTestNode(t, uri)
		defer func() { _ = backend.Close() }()

		var (
			client = backend.Attach()
			ctx    = context.TODO()
			result hexutil.Uint64
		)
		defer client.Close()

		err := client.CallContext(ctx, &result, "mev_blockNumber")
		require.NoError(t, err)
		assert.Equal(t, uint64(100500), uint64(result))
	})

	t.Run("eth namespace over rpc", func(t *testing.T) {
		// run RPC server that emulates a miner the Sentry is forwarding requests to
		api := new(MockMevAPI)
		uri, stop := newRpcServer("mev", api)
		defer stop()

		// run test Ethereum Node Backend
		backend, _ := newTestNode(t, uri)
		defer func() { _ = backend.Close() }()

		var (
			client = backend.Attach()
			ctx    = context.TODO()
			result hexutil.Uint64
		)
		defer client.Close()

		err := client.CallContext(ctx, &result, "eth_blockNumber")
		require.NoError(t, err)
		assert.Equal(t, uint64(100500), uint64(result))
	})

	t.Run("call mev and eth implementations over console", func(t *testing.T) {
		// run RPC server that emulates a miner the Sentry is forwarding requests to
		api := new(MockMevAPI)
		uri, stop := newRpcServer("mev", api)
		defer stop()

		backend, _ := newTestNode(t, uri)
		defer func() { _ = backend.Close() }()

		var (
			client   = backend.Attach()
			printer  = new(bytes.Buffer)
			prompter = &hookedPrompter{scheduler: make(chan string)}
		)
		defer client.Close()

		cli, err := console.New(console.Config{
			DataDir:  backend.DataDir(),
			Client:   client,
			Printer:  printer,
			Prompter: prompter,
		})
		require.NoError(t, err)

		// mev_blockNumber should work similar to eth.blockNumber,
		// and be acceptable from console like a property.
		cli.Evaluate("[mev.blockNumber, eth.blockNumber]")
		assert.Equal(t, "[100500, 100500]\n", printer.String())
	})
}

func TestProposalBlock(t *testing.T) {
	// run RPC server that emulates a miner the Sentry is forwarding requests to
	api := new(MockMevAPI)
	uri, stop := newRpcServer("mev", api)
	defer stop()

	// run test Ethereum Node Backend
	backend, _ := newTestNode(t, uri)
	defer func() { _ = backend.Close() }()

	// Attach to the currently testing Node
	client := backend.Attach()
	defer client.Close()

	var (
		ctx    = context.TODO()
		result interface{}
		arg    = ethapi.ProposedBlockArgs{
			MEVRelay:         "mev-something",
			BlockNumber:      42,
			PrevBlockHash:    common.HexToHash("0xDEADBEEF"),
			BlockReward:      big.NewInt(42),
			GasLimit:         1000,
			GasUsed:          500,
			Payload:          []any{4, 8, 15, 16, 23, 42},
			UnRevertedHashes: []common.Hash{common.HexToHash("0xFACEXD")},
		}
	)

	if err := client.CallContext(ctx, &result, "mev_proposedBlock", arg); err != nil {
		t.Fatalf("expected to not have error, but got: %s", err)
	}

	expectedJsonArg := `{
		"blockNumber":"0x2a",
		"prevBlockHash":"0x00000000000000000000000000000000000000000000000000000000deadbeef",
		"unRevertedHashes":["0x000000000000000000000000000000000000000000000000000000000000face"],
		"blockReward":42,
		"gasLimit":1000,
		"gasUsed":500,
		"payload":[4,8,15,16,23,42],
		"mevRelay":"mev-something"
	}`

	require.NotEmpty(t, api.ProposedBlockCalledArg)
	assert.JSONEq(t, expectedJsonArg, api.ProposedBlockCalledArg)
}

func TestRegisterValidator(t *testing.T) {
	// run RPC server that emulates a builder the Sentry is forwarding requests to
	api := new(MockMevAPI)
	uri, stop := newRpcServer("mev", api)
	defer stop()

	// run test Ethereum Node Backend
	backend, _ := newTestNode(t, uri)
	defer func() { _ = backend.Close() }()

	// Attach to the currently testing Node
	client := backend.Attach()
	defer client.Close()

	var (
		ctx    = context.TODO()
		result interface{}
		arg    = ethapi.RegisterValidatorArgs{
			Data:       []byte{0xDE, 0xAD, 0xBE, 0xEF},
			Signature:  []byte("trust-me"),
			IsSentry:   true,
			Namespace:  "mev",
			CommitHash: "some-hash",
			GasCeil:    1000,
		}
	)

	if err := client.CallContext(ctx, &result, "mev_registerValidator", arg); err != nil {
		t.Fatalf("expected to not have error, but got: %s", err)
	}

	expectedJsonArg := `{
		"data":"0xdeadbeef",
		"signature":"0x74727573742d6d65",
		"isSentry":true,
		"gasCeil":1000,
		"namespace":"mev",
		"commitHash":"some-hash"
	}`

	require.NotEmpty(t, api.RegisterValidatorArg)
	assert.JSONEq(t, expectedJsonArg, api.RegisterValidatorArg)
}
