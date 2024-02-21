package miner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAcceptRelayMap(t *testing.T) {
	t.Run("MarshalUnmarshal", func(t *testing.T) {
		originalRelays := []string{"relay1", "relay2", "relay3"}
		arm := NewAcceptRelayMap(originalRelays...)

		// Marshal
		text, err := arm.MarshalText()
		require.NoError(t, err)

		// Unmarshal
		arm2 := NewAcceptRelayMap()
		require.NoError(t, arm2.UnmarshalText(text))
		require.Equal(t, arm.len, arm2.len)

		for _, relay := range append(originalRelays, "relay4") {
			require.Equal(t, arm.Accept(relay), arm2.Accept(relay))
		}
	})

	t.Run("Empty", func(t *testing.T) {
		arm := NewAcceptRelayMap()

		// Marshal and Unmarshal an empty map
		text, err := arm.MarshalText()
		require.NoError(t, err)

		arm2 := NewAcceptRelayMap()
		require.NoError(t, arm2.UnmarshalText(text))
		require.Equal(t, arm.len, arm2.len)
		require.Equal(t, arm.relays.Size(), arm2.relays.Size())
		require.Equal(t, arm.relays, arm2.relays)
	})

	t.Run("UnmarshalError", func(t *testing.T) {
		// Check if an error is returned due to invalid JSON
		require.Error(t, new(AcceptRelayMap).UnmarshalText([]byte("not a valid json")))
	})
}
