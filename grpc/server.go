package grpc

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"

	pb "github.com/ethereum/go-ethereum/grpc/protobuf"
	"github.com/ethereum/go-ethereum/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	windowSize = 128 * 1024
	// bufferSize determines how much data can be batched before doing a write
	// on the wire. Zero or negative values will disable the write buffer such that each
	// write will be on underlying connection.
	bufferSize = 0
)

type API struct {
	proposer   *Proposer
	listenAddr string
	authHeader string
	server     *grpc.Server
}

func NewAPI(proposer *Proposer, listenAddr, nodeID, secret string) *API {

	var authHeader string
	if nodeID != "" && secret != "" {
		authHeader = encodeNodeSecret(nodeID, secret)
	}

	return &API{
		proposer:   proposer,
		listenAddr: listenAddr,
		authHeader: authHeader,
	}
}

func (a *API) Start() error {
	a.run()
	return nil
}

func (a *API) Stop() {
	server := a.server
	if server != nil {
		a.server.Stop()
	}
}

func (a *API) run() {
	listener, err := net.Listen("tcp", a.listenAddr)
	if err != nil {
		log.Crit("failed to run grpc server", "err", err)
	}

	serverOptions := []grpc.ServerOption{
		grpc.WriteBufferSize(bufferSize),
		grpc.InitialConnWindowSize(windowSize),
		grpc.UnaryInterceptor(a.authenticate),
	}

	a.server = grpc.NewServer(serverOptions...)
	//	pb.RegisterGatewayServer(ggs.server, ggs.gateway)
	pb.RegisterProposerServer(a.server, a.proposer)
	log.Info("proposer GRPC server is listing", "address", a.listenAddr)
	if err := a.server.Serve(listener); err != nil {
		log.Crit("failed to serve: %v", err)
	}
}

func (a *API) authenticate(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if a.authHeader != "" {
		auth, err := readAuthMetadata(ctx)
		if err != nil {
			return nil, err
		}

		if auth != a.authHeader {
			return nil, errors.New("incorrect auth header provided")
		}
	}
	return handler(ctx, req)
}

// readAuthMetadata reads auth info from the GRPC connection context
func readAuthMetadata(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("could not read metadata from context")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", errors.New("empty auth header provided")
	}
	return values[0], nil
}

// EncodeUserSecret produces a base64 encoded auth header of a user and secret
func encodeNodeSecret(user string, secret string) string {
	data := fmt.Sprintf("%v:%v", user, secret)
	return base64.StdEncoding.EncodeToString([]byte(data))
}
