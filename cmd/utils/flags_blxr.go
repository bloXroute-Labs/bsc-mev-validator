package utils

import (
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
)

var (
	HTTPSecuredIPPortFlag = &cli.IntFlag{
		Name:     "http.securedipport",
		Usage:    "HTTP-RPC server secured by IP listening port",
		Value:    node.DefaultHTTPSecuredIPPort,
		Category: flags.APICategory,
	}
	HTTPSecuredIPAllowedIPsFlag = &cli.StringFlag{
		Name:     "http.securedipvallowedipss",
		Usage:    "Comma separated list of IPs from which to accept requests (server enforced). Accepts '*' wildcard.",
		Value:    strings.Join(node.DefaultConfig.HTTPSecuredIPAllowedIPs, ","),
		Category: flags.APICategory,
	}
	HTTPSecuredIPApiFlag = &cli.StringFlag{
		Name:     "http.securedipapi",
		Usage:    "Comma separated list of API's offered over the HTTP-RPC secured by IP interface",
		Value:    "",
		Category: flags.APICategory,
	}

	HTTPSecuredFlags = []cli.Flag{
		HTTPSecuredIPPortFlag,
		HTTPSecuredIPAllowedIPsFlag,
		HTTPSecuredIPApiFlag,
	}
)

// setHTTPSecuredIP creates the HTTP MEV RPC listener interface string from the set
// command line flags, returning empty if the HTTP endpoint is disabled.
func setHTTPSecuredIP(ctx *cli.Context, cfg *node.Config) {
	if ctx.Bool(HTTPEnabledFlag.Name) {
		if cfg.HTTPHost == "" {
			cfg.HTTPHost = "127.0.0.1"
		}
		if ctx.IsSet(HTTPListenAddrFlag.Name) {
			cfg.HTTPHost = ctx.String(HTTPListenAddrFlag.Name)
		}
	}

	if ctx.IsSet(HTTPSecuredIPPortFlag.Name) {
		cfg.HTTPSecuredIPPort = ctx.Int(HTTPSecuredIPPortFlag.Name)
	}

	if ctx.IsSet(HTTPCORSDomainFlag.Name) {
		cfg.HTTPCors = SplitAndTrim(ctx.String(HTTPCORSDomainFlag.Name))
	}

	if ctx.IsSet(HTTPSecuredIPApiFlag.Name) {
		cfg.HTTPSecuredIPModules = SplitAndTrim(ctx.String(HTTPSecuredIPApiFlag.Name))
	}

	if ctx.IsSet(HTTPSecuredIPAllowedIPsFlag.Name) {
		cfg.HTTPSecuredIPAllowedIPs = SplitAndTrim(ctx.String(HTTPSecuredIPAllowedIPsFlag.Name))
	}

	if ctx.IsSet(HTTPPathPrefixFlag.Name) {
		cfg.HTTPPathPrefix = ctx.String(HTTPPathPrefixFlag.Name)
	}

	if ctx.IsSet(AllowUnprotectedTxs.Name) {
		cfg.AllowUnprotectedTxs = ctx.Bool(AllowUnprotectedTxs.Name)
	}
}

var (
	MinerMEVRelaysFlag = &cli.StringSliceFlag{
		Name:     "miner.mevrelays",
		Usage:    "Destinations to register the validator each epoch. The miner will accept proposed blocks from these urls, if they are profitable.",
		Category: flags.MinerCategory,
	}
	MinerMEVProposedBlockUriFlag = &cli.StringFlag{
		Name:     "miner.mevproposedblockuri",
		Usage:    "The uri MEV relays should send the proposedBlock to.",
		Category: flags.MinerCategory,
	}
	MinerMEVProposedBlockNamespaceFlag = &cli.StringFlag{
		Name:     "miner.mevproposedblocknamespace",
		Usage:    "The namespace implements the proposedBlock function (default = eth). ",
		Value:    "eth",
		Category: flags.MinerCategory,
	}

	MinerMEVFlags = []cli.Flag{
		MinerMEVRelaysFlag,
		MinerMEVProposedBlockUriFlag,
		MinerMEVProposedBlockNamespaceFlag,
	}
)

func setMEV(ctx *cli.Context, stack *node.Node, cfg *miner.Config) {
	if ctx.IsSet(MinerMEVRelaysFlag.Name) {
		cfg.MEVRelays = ctx.StringSlice(MinerMEVRelaysFlag.Name)
	}

	if len(cfg.MEVRelays) == 0 {
		return
	}

	keystores := stack.AccountManager().Backends(keystore.KeyStoreType)
	if len(keystores) == 0 {
		return
	}

	ks, ok := keystores[0].(*keystore.KeyStore)
	if !ok {
		return
	}

	if ctx.IsSet(MinerMEVProposedBlockUriFlag.Name) {
		cfg.ProposedBlockUri = ctx.String(MinerMEVProposedBlockUriFlag.Name)
	}

	if cfg.ProposedBlockNamespace == "" && ctx.IsSet(MinerMEVProposedBlockNamespaceFlag.Name) {
		cfg.ProposedBlockNamespace = ctx.String(MinerMEVProposedBlockNamespaceFlag.Name)
	}

	account, err := ks.Find(accounts.Account{Address: cfg.Etherbase})
	if err != nil {
		Fatalf("Could not find the validator public address %v to sign the registerValidator message, %v", cfg.Etherbase, err)
	}

	registerHash := accounts.TextHash([]byte(cfg.ProposedBlockUri))
	passwordList := MakePasswordList(ctx)
	if passwordList == nil {
		cfg.RegisterValidatorSignedHash, err = ks.SignHash(account, registerHash)
		if err != nil {
			Fatalf("Failed sign registerValidator message unlocked with error: %v", err)
		}
	} else {
		passwordFound := false
		for _, password := range passwordList {
			cfg.RegisterValidatorSignedHash, err = ks.SignHashWithPassphrase(account, password, registerHash)
			if err == nil {
				passwordFound = true
				break
			}
		}
		if !passwordFound {
			Fatalf("Failed sign registerValidator message with passphrase with error")
		}
	}

	signature := make([]byte, crypto.SignatureLength)
	copy(signature, cfg.RegisterValidatorSignedHash)
	// verify the validator public address used to sign the registerValidator message
	if len(signature) != crypto.SignatureLength {
		Fatalf("signature used to sign registerValidator must be %d bytes long", crypto.SignatureLength)
	}

	if signature[crypto.RecoveryIDOffset] == 27 || signature[crypto.RecoveryIDOffset] == 28 {
		signature[crypto.RecoveryIDOffset] -= 27 // Transform yellow paper V from 27/28 to 0/1
	}

	if signature[crypto.RecoveryIDOffset] != 0 && signature[crypto.RecoveryIDOffset] != 1 {
		Fatalf("invalid Ethereum signature of the registerValidator (V is not 0, or 1, or 27 or 28), it is %v", cfg.RegisterValidatorSignedHash[crypto.RecoveryIDOffset])
	}

	rpk, err := crypto.SigToPub(registerHash, signature)
	if err != nil {
		Fatalf("Failed to get validator public address from the registerValidator signed message %v", err)
	}

	addr := crypto.PubkeyToAddress(*rpk)
	if addr != account.Address {
		Fatalf("Validator public Etherbase %v was not used to sign the registerValidator %v", account.Address, addr)
	}
}
