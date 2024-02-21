package blxerr

import (
	"math/big"
	"testing"
)

func TestProposedBlockLessProfitableErr_Error(t *testing.T) {
	type fields struct {
		RewardThreshold *big.Int
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			"with reward value",
			fields{big.NewInt(10_000)},
			"block cannot be accepted. Reward threshold is 10000",
		},

		{
			"without reward value",
			fields{nil},
			"block cannot be accepted. Reward threshold is <nil>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ProposedBlockLessProfitableErr{
				RewardThreshold: tt.fields.RewardThreshold,
			}
			if got := err.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}
