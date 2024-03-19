package blxerr

import (
	"fmt"
	"math/big"
)

// ProposedBlockLessProfitableErr happens when a proposed block cannot be accepted because of the block is less profitable
// than a cached block.
type ProposedBlockLessProfitableErr struct {
	RewardThreshold *big.Int
}

func (err ProposedBlockLessProfitableErr) Error() string {
	// do not change error message as it's sent to validator's clients and builders parse it!
	return fmt.Sprintf("block cannot be accepted. Reward threshold is %d", err.RewardThreshold)
}
