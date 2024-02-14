package utils

import (
	"strings"

	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/node"
	"github.com/urfave/cli/v2"
)

var (
	SentryMinerUriFlag = &cli.StringFlag{
		Name:  "sentry.mineruri",
		Usage: "Uri used by the proxy forwarding blocks from the relay to the validator",
	}
	SentryRelaysUriFlag = &cli.StringSliceFlag{
		Name:  "sentry.relaysuri",
		Usage: "Slice of MEV relay uris sending the registration from the validator node",
	}

	SentryFlags = []cli.Flag{
		SentryMinerUriFlag,
		SentryRelaysUriFlag,
	}
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

// setSentry configures the sentry settings from the command line flags.
func setSentry(ctx *cli.Context, cfg *ethconfig.Config) {
	cfg.SentryMinerUri = ctx.String(SentryMinerUriFlag.Name)
	cfg.SentryRelaysUri = ctx.StringSlice(SentryRelaysUriFlag.Name)
}
