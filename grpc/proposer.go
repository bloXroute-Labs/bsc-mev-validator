package grpc

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	pb "github.com/ethereum/go-ethereum/grpc/protobuf"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// timestamp format
	timestampFormat = "2006-01-02 15:04:05.000000"
)

var _ pb.ProposerServer = (*Proposer)(nil)

type Proposer struct {
	backend ethapi.Backend
	pb.UnimplementedProposerServer
}

func NewProposer(backend ethapi.Backend) *Proposer {
	return &Proposer{backend: backend}
}

func (p *Proposer) ProposeBlock(ctx context.Context, in *pb.ProposeBlockRequest) (*pb.ProposeBlockResponse, error) {

	if len(in.Payload) == 0 {
		return nil, errors.New("proposed block missing txs")
	}
	if in.BlockNumber == 0 {
		return nil, errors.New("proposed block missing blockNumber")
	}
	args := &ethapi.ProposedBlockArgs{
		MEVRelay:      in.GetMevRelay(),
		BlockNumber:   rpc.BlockNumber(in.GetBlockNumber()),
		PrevBlockHash: common.HexToHash(in.GetPrevBlockHash()),
		BlockReward:   new(big.Int).SetUint64(in.GetBlockReward()),
		GasLimit:      in.GetGasLimit(),
		GasUsed:       in.GetGasLimit(),
		Payload:       in.GetPayload(),
	}

	out, err := p.backend.ProposedBlock(ctx, args, in.GetNamespace())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return out.(*pb.ProposeBlockResponse), nil
}
