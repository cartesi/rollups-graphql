// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This package contains the main function that executes the cartesi-rollups-graphql command.
package main

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/carlmjohnson/versioninfo"
	"github.com/cartesi/rollups-graphql/v2/pkg/bootstrap"
	"github.com/cartesi/rollups-graphql/v2/pkg/commons"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/joho/godotenv"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

var (
	MAX_FILE_SIZE uint64 = 1_440_000 // 1,44 MB
)

var startupMessage = `
GraphQL running at http://localhost:HTTP_PORT/graphql
Press Ctrl+C to stop the node
`

var tempFromBlockL1 uint64

var cmd = &cobra.Command{
	Use:     "cartesi-rollups-graphql [flags] [-- application [args]...]",
	Short:   "cartesi-rollups-graphql is a development node for Cartesi Rollups",
	Run:     run,
	Version: versioninfo.Short(),
}

var CompletionCmd = &cobra.Command{
	Use:                   "completion",
	Short:                 "Generate shell completion scripts",
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cobra.CheckErr(cmd.Root().GenBashCompletion(os.Stdout))
		case "zsh":
			cobra.CheckErr(cmd.Root().GenZshCompletion(os.Stdout))
		case "fish":
			cobra.CheckErr(cmd.Root().GenFishCompletion(os.Stdout, true))
		case "powershell":
			cobra.CheckErr(cmd.Root().GenPowerShellCompletion(os.Stdout))
		}
	},
}

var (
	debug bool
	color bool
	opts  = bootstrap.NewBootstrapOpts()
)

func ArrBytesAttr(key string, v []byte) slog.Attr {
	var str string
	for _, b := range v {
		s := fmt.Sprintf("%02x", b)
		str += s
	}
	return slog.String(key, str)
}

func CheckIfValidSize(size uint64) error {
	if size > MAX_FILE_SIZE {
		return fmt.Errorf("file size is too big %d bytes", size)
	}

	return nil
}

func init() {
	// contracts-*
	cmd.Flags().StringVar(&opts.ApplicationAddress, "contracts-application-address",
		opts.ApplicationAddress, "Application contract address")

	// enable-*
	cmd.Flags().BoolVarP(&debug, "enable-debug", "d", false, "If set, enable debug output")
	cmd.Flags().BoolVar(&color, "enable-color", true, "If set, enables logs color")

	cmd.Flags().DurationVar(&opts.TimeoutWorker, "timeout-worker", opts.TimeoutWorker, "Timeout for workers. Example: cartesi-rollups-graphql --timeout-worker 30s")

	// disable-*

	// http-*
	cmd.Flags().StringVar(&opts.HttpAddress, "http-address", opts.HttpAddress,
		"HTTP address used by cartesi-rollups-graphql to serve its APIs")
	cmd.Flags().IntVar(&opts.HttpPort, "http-port", opts.HttpPort,
		"HTTP port used by cartesi-rollups-graphql to serve its external APIs")

	// database file
	cmd.Flags().StringVar(&opts.SqliteFile, "sqlite-file", opts.SqliteFile,
		"The sqlite file to load the state")

	cmd.Flags().Uint64VarP(&tempFromBlockL1, "from-l1-block", "", 0, "The beginning of the queried range for events")

	cmd.Flags().StringVar(&opts.DbImplementation, "db-implementation", opts.DbImplementation,
		"DB to use. PostgreSQL or SQLite")

	cmd.Flags().BoolVar(&opts.DisableSync, "disable-sync", opts.DisableSync, "If set disable data synchronization")
}

func deprecatedWarningCmd(cmd *cobra.Command, flag string, replacement string) {
	if cmd.Flags().Changed(flag) {
		slog.WarnContext(cmd.Context(), fmt.Sprintf("The '%s' flag is deprecated. %s", flag, replacement))
	}
}

func deprecatedFlags(cmd *cobra.Command) {
	checkAndSetFlag(cmd, "contracts-application-address", func(val string) { opts.ApplicationAddress = val }, "APPLICATION_ADDRESS")
	checkAndSetFlag(cmd, "enable-debug", func(val string) { debug = cast.ToBool(val) }, "GRAPHQL_DEBUG")
	checkAndSetFlag(cmd, "enable-color", func(val string) { color = cast.ToBool(val) }, "COLOR")
	checkAndSetFlag(cmd, "timeout-worker", func(val string) { opts.TimeoutWorker, _ = time.ParseDuration(val) }, "TIMEOUT_WORKER")
	checkAndSetFlag(cmd, "http-address", func(val string) { opts.HttpAddress = val }, "HTTP_ADDRESS")
	checkAndSetFlag(cmd, "http-port", func(val string) { opts.HttpPort = cast.ToInt(val) }, "HTTP_PORT")
	checkAndSetFlag(cmd, "sqlite-file", func(val string) { opts.SqliteFile = val }, "SQLITE_FILE")
	checkAndSetFlag(cmd, "from-l1-block", func(val string) { tempFromBlockL1 = cast.ToUint64(val) }, "FROM_BLOCK_L1")
	checkAndSetFlag(cmd, "db-implementation", func(val string) { opts.DbImplementation = val }, "DB_IMPLEMENTATION")
	checkAndSetFlag(cmd, "disable-sync", func(val string) { opts.DisableSync = cast.ToBool(val) }, "DISABLE_SYNC")
}

/**
 * Check if the flag is set and set the value from the environment variable
 */
func checkAndSetFlag(cmd *cobra.Command, flagName string, setOptEnv func(string), flagEnv string) {
	val, isEnvVarPresent := os.LookupEnv(flagEnv)
	if isEnvVarPresent {
		setOptEnv(val)
	}
	deprecatedMsg := fmt.Sprint("Please use ", flagEnv, " instead.")
	deprecatedWarningCmd(cmd, flagName, deprecatedMsg)
}

func run(cmd *cobra.Command, args []string) {
	LoadEnv(cmd.Context())
	ctx := cmd.Context()
	startTime := time.Now()

	// setup log
	levelDebug := slog.LevelInfo
	if debug {
		levelDebug = slog.LevelDebug
	}
	commons.ConfigureLogForProduction(levelDebug, color)

	// check args
	checkEthAddress(cmd, "address-input-box")
	checkEthAddress(cmd, "address-application")
	deprecatedFlags(cmd)

	// handle signals with notify context
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	startMessage := startupMessage

	var inspectMessage string

	// start cartesi-rollups-graphql
	ready := make(chan struct{}, 1)
	go func() {
		select {
		case <-ready:
			msg := strings.Replace(
				startMessage,
				"\nINSPECT_MESSAGE",
				inspectMessage, -1)
			msg = strings.Replace(
				msg,
				"HTTP_PORT",
				fmt.Sprint(opts.HttpPort), -1)
			fmt.Println(msg)
			slog.InfoContext(ctx, "cartesi-rollups-graphql: ready", "after", time.Since(startTime))
		case <-ctx.Done():
		}
	}()
	var err error = bootstrap.NewSupervisorGraphQL(ctx, opts).Start(ctx, ready)
	cobra.CheckErr(err)
}

//go:embed .env
var envBuilded string

// LoadEnv from embedded .env file
func LoadEnv(ctx context.Context) {
	currentEnv := map[string]bool{}
	rawEnv := os.Environ()
	for _, rawEnvLine := range rawEnv {
		key := strings.Split(rawEnvLine, "=")[0]
		currentEnv[key] = true
	}

	parse, err := godotenv.Unmarshal(envBuilded)
	cobra.CheckErr(err)

	for k, v := range parse {
		if !currentEnv[k] {
			slog.DebugContext(ctx, "env: setting env", "key", k, "value", v)
			err := os.Setenv(k, v)
			cobra.CheckErr(err)
		} else {
			slog.DebugContext(ctx, "env: skipping env", "key", k)
		}
	}

	slog.DebugContext(ctx, "env: loaded")
}

func main() {
	cmd.AddCommand(CompletionCmd)
	cobra.CheckErr(cmd.Execute())
}

func exitf(ctx context.Context, format string, args ...any) {
	err := fmt.Sprintf(format, args...)
	slog.ErrorContext(ctx, "configuration error", "error", err)
	os.Exit(1)
}

func checkEthAddress(cmd *cobra.Command, varName string) {
	if cmd.Flags().Changed(varName) {
		ctx := cmd.Context()
		value, err := cmd.Flags().GetString(varName)
		cobra.CheckErr(err)
		bytes, err := hexutil.Decode(value)
		if err != nil {
			exitf(ctx, "invalid address for --%v: %v", varName, err)
		}
		if len(bytes) != common.AddressLength {
			exitf(ctx, "invalid address for --%v: wrong length", varName)
		}
	}
}
