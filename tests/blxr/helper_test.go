package blxr

import (
	"errors"
	"math/big"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/console/prompt"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	testKey, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testAddr    = crypto.PubkeyToAddress(testKey.PublicKey)
	testSlot    = common.HexToHash("0xdeadbeef")
	testValue   = crypto.Keccak256Hash(testSlot[:])
	testBalance = big.NewInt(2e15)
)

func generateTestChain() (*core.Genesis, []*types.Block) {
	genesis := &core.Genesis{
		Config:    params.AllEthashProtocolChanges,
		Alloc:     core.GenesisAlloc{testAddr: {Balance: testBalance, Storage: map[common.Hash]common.Hash{testSlot: testValue}}},
		ExtraData: []byte("test genesis"),
		Timestamp: 9000,
	}
	generate := func(i int, g *core.BlockGen) {
		g.OffsetTime(5)
		g.SetExtra([]byte("test"))
	}
	_, blocks, _ := core.GenerateChainWithGenesis(genesis, ethash.NewFaker(), 1, generate)
	blocks = append([]*types.Block{genesis.ToBlock()}, blocks...)
	return genesis, blocks
}

func newRpcServer(namespace string, service interface{}) (uri string, stop func()) {
	server := rpc.NewServer()
	if err := server.RegisterName(namespace, service); err != nil {
		panic("can't register service on namespace")
	}

	ts := httptest.NewServer(server)

	return ts.URL, func() {
		ts.Close()
		server.Stop()
	}
}

func newTestNode(t *testing.T, sentryBackendUri string) (*node.Node, []*types.Block) {
	// Generate test chain.
	workspace := t.TempDir()
	genesis, blocks := generateTestChain()
	// Create node
	n, err := node.New(&node.Config{DataDir: workspace})
	if err != nil {
		t.Fatalf("can't create new node: %v", err)
	}

	// Create Ethereum Service
	config := &ethconfig.Config{
		Genesis:         genesis,
		SentryMinerUri:  sentryBackendUri,
		SentryRelaysUri: []string{sentryBackendUri},
	}
	ethservice, err := eth.New(n, config)
	if err != nil {
		t.Fatalf("can't create new ethereum service: %v", err)
	}

	// Import the test chain.
	if err := n.Start(); err != nil {
		t.Fatalf("can't start test node: %v", err)
	}
	if _, err := ethservice.BlockChain().InsertChain(blocks[1:]); err != nil {
		t.Fatalf("can't import test blocks: %v", err)
	}

	return n, blocks
}

type hookedPrompter struct {
	scheduler chan string
}

func (p *hookedPrompter) PromptInput(prompt string) (string, error) {
	// Send the prompt to the tester
	select {
	case p.scheduler <- prompt:
	case <-time.After(time.Second):
		return "", errors.New("prompt timeout")
	}
	// Retrieve the response and feed to the console
	select {
	case input := <-p.scheduler:
		return input, nil
	case <-time.After(time.Second):
		return "", errors.New("input timeout")
	}
}

func (p *hookedPrompter) PromptPassword(prompt string) (string, error) {
	return "", errors.New("not implemented")
}
func (p *hookedPrompter) PromptConfirm(prompt string) (bool, error) {
	return false, errors.New("not implemented")
}
func (p *hookedPrompter) SetHistory(history []string)                     {}
func (p *hookedPrompter) AppendHistory(command string)                    {}
func (p *hookedPrompter) ClearHistory()                                   {}
func (p *hookedPrompter) SetWordCompleter(completer prompt.WordCompleter) {}
