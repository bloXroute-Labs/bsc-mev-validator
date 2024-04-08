package blxr

import (
	"bytes"
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/console"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockNumber(t *testing.T) {
	t.Run("mev namespace over rpc", func(t *testing.T) {
		// run test Ethereum Node Backend
		backend, _ := newTestNode(t)
		defer func() { _ = backend.Close() }()

		var (
			client = backend.Attach()
			ctx    = context.TODO()
			result hexutil.Uint64
		)
		defer client.Close()

		err := client.CallContext(ctx, &result, "mev_blockNumber")
		require.NoError(t, err)
		assert.Equal(t, uint64(1), uint64(result))
	})

	t.Run("eth namespace over rpc", func(t *testing.T) {
		// run test Ethereum Node Backend
		backend, _ := newTestNode(t)
		defer func() { _ = backend.Close() }()

		var (
			client = backend.Attach()
			ctx    = context.TODO()
			result hexutil.Uint64
		)
		defer client.Close()

		err := client.CallContext(ctx, &result, "eth_blockNumber")
		require.NoError(t, err)
		assert.Equal(t, uint64(1), uint64(result))
	})

	t.Run("call mev and eth implementations over console", func(t *testing.T) {
		backend, _ := newTestNode(t)
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
		assert.Equal(t, "[1, 1]\n", printer.String())
	})
}
