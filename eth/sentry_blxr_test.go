package eth

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRelayClient struct {
	calls   int
	failErr error
	match   func(gotMethod string, gotArg interface{})
}

func (m *mockRelayClient) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	m.calls++
	if m.match != nil {
		m.match(method, args[0])
	}
	return m.failErr
}

type mockMinerClient struct {
	called   bool
	failErr  error
	callback func(method string, args ...json.RawMessage) json.RawMessage
}

func (m *mockMinerClient) Forward(_ context.Context, result interface{}, method string, args ...json.RawMessage) error {
	m.called = true
	var response json.RawMessage
	if m.callback != nil {
		response = m.callback(method, args...)
	}

	if err := json.Unmarshal(response, result); err != nil {
		panic(err)
	}

	return m.failErr
}

func TestSentryProxy_RegisterValidator(t *testing.T) {
	t.Run("do not fail if relays is empty", func(t *testing.T) {
		var (
			proxy = &SentryProxy{}
			arg   = ethapi.RegisterValidatorArgs{
				Data:       []byte{0xDE, 0xAD, 0xF0, 0x0D},
				Signature:  []byte("trust-me"),
				IsSentry:   true,
				Namespace:  "mev",
				CommitHash: "some-hash",
				GasCeil:    1000,
			}
		)

		err := proxy.RegisterValidator(context.TODO(), arg)
		require.NoError(t, err)
	})

	t.Run("forward request to each rely", func(t *testing.T) {
		var (
			arg = ethapi.RegisterValidatorArgs{
				Data:       []byte{0xDE, 0xAD, 0xF0, 0x0D},
				Signature:  []byte("trust-me"),
				IsSentry:   true,
				Namespace:  "mev",
				CommitHash: "some-hash",
				GasCeil:    1000,
			}
			client = &mockRelayClient{
				match: func(gotMethod string, gotArg interface{}) {
					assert.Equal(t, "mev_registerValidator", gotMethod)
					assert.Equal(t, arg, gotArg)
				},
			}
			proxy = &SentryProxy{
				relays: []ContextCaller{client, client, client},
			}
		)

		err := proxy.RegisterValidator(context.TODO(), arg)
		require.NoError(t, err)

		assert.Equal(t, len(proxy.relays), client.calls)
	})

	t.Run("when a call fails", func(t *testing.T) {
		var (
			client            = &mockRelayClient{}
			clientWithFailure = &mockRelayClient{failErr: errors.New("oops")}

			proxy = &SentryProxy{
				relays: []ContextCaller{client, clientWithFailure, client, clientWithFailure, client},
			}
			arg = ethapi.RegisterValidatorArgs{
				Data:       []byte{0xDE, 0xAD, 0xF0, 0x0D},
				Signature:  []byte("trust-me"),
				IsSentry:   true,
				Namespace:  "mev",
				CommitHash: "some-hash",
				GasCeil:    1000,
			}
		)

		err := proxy.RegisterValidator(context.TODO(), arg)
		assert.Error(t, err)

		successfulCalls := len(proxy.relays) - 2 // 2 calls should fail
		assert.Equal(t, successfulCalls, client.calls)
	})
}

func TestSentryProxy_ProposedBlock(t *testing.T) {
	t.Run("do not fail if miner is empty", func(t *testing.T) {
		var (
			proxy = &SentryProxy{}
			arg   = `{"something":"foobar"}`
		)

		result, err := proxy.ProposedBlock(context.TODO(), json.RawMessage(arg))
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("forward to miner", func(t *testing.T) {
		var (
			arg    = json.RawMessage(`{"something":"foobar"}`)
			client = &mockMinerClient{
				callback: func(method string, args ...json.RawMessage) json.RawMessage {
					assert.Equal(t, "mev_proposedBlock", method)
					assert.Equal(t, arg, args[0])

					return json.RawMessage(`"some useful response from miner"`)
				},
			}
			proxy = &SentryProxy{
				miner: client,
			}
		)

		result, err := proxy.ProposedBlock(context.TODO(), arg)
		assert.Nil(t, err)
		assert.Equal(t, "some useful response from miner", result)
		assert.True(t, client.called)
	})
}

func TestSentryProxy_BlockNumber(t *testing.T) {
	t.Run("do not fail if miner is empty", func(t *testing.T) {
		var (
			proxy = &SentryProxy{}
		)

		result, err := proxy.BlockNumber(context.TODO())
		require.NoError(t, err)
		assert.Equal(t, uint64(0), uint64(result))
	})

	t.Run("forward to miner", func(t *testing.T) {
		var (
			client = &mockMinerClient{
				callback: func(method string, args ...json.RawMessage) json.RawMessage {
					assert.Equal(t, "mev_blockNumber", method)
					assert.Nil(t, args)
					return json.RawMessage(`"0xFF"`)
				},
			}
			proxy = &SentryProxy{
				miner: client,
			}
		)

		result, err := proxy.BlockNumber(context.TODO())
		assert.Nil(t, err)
		assert.Equal(t, uint64(255), uint64(result))
		assert.True(t, client.called)
	})
}
