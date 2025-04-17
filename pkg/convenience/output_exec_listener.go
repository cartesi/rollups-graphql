package convenience

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/cartesi/rollups-graphql/v2/pkg/contracts"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience/services"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type OutputExecListener struct {
	Provider           string
	ApplicationAddress common.Address
	EventName          string
	ConvenienceService *services.ConvenienceService
	FromBlock          *big.Int
}

func NewExecListener(
	provider string,
	applicationAddress common.Address,
	convenienceService *services.ConvenienceService,
	fromBlock *big.Int,
) OutputExecListener {
	return OutputExecListener{
		FromBlock:          fromBlock,
		ConvenienceService: convenienceService,
		Provider:           provider,
		ApplicationAddress: applicationAddress,
		EventName:          "OutputExecuted",
	}
}

// on event callback
func (x OutputExecListener) OnEvent(
	eventValues []interface{},
	timestamp,
	blockNumber uint64,
) error {
	if len(eventValues) != 1 {
		return fmt.Errorf("wrong event values length != 1")
	}
	voucherId, ok := eventValues[0].(*big.Int)
	if !ok {
		return fmt.Errorf("cannot cast voucher id to big.Int")
	}

	// Extract voucher and input using bit masking and shifting
	var bitsToShift uint = 128
	var maxHexBytes uint64 = 0xFFFFFFFFFFFFFFFF
	bitMask := new(big.Int).SetUint64(maxHexBytes)
	voucher := new(big.Int).Rsh(voucherId, bitsToShift)
	input := new(big.Int).And(voucherId, bitMask)

	// Print the extracted voucher and input
	slog.Debug("Decoded voucher params",
		"voucher", voucher,
		"input", input,
		"blockNumber", blockNumber,
	)

	// Print decoded event data
	slog.Debug("Voucher Executed", "voucherId", voucherId.String())

	ctx := context.Background()
	return x.ConvenienceService.UpdateExecuted(ctx, input.Uint64(), voucher.Uint64(), true)
}

// String implements supervisor.Worker.
func (x OutputExecListener) String() string {
	return "OutputExecListener"
}

func (x OutputExecListener) Start(ctx context.Context, ready chan<- struct{}) error {
	var delay time.Duration = 5 * time.Second
	slog.Info("Connecting to", "provider", x.Provider)

	var client *ethclient.Client
	var err error

	for {
		ctxDial, cancel := context.WithCancel(ctx)
		defer cancel()

		client, err = ethclient.DialContext(ctxDial, x.Provider)
		if err == nil {
			break
		}

		slog.Error("execlistener: dial: ", "error", err)
		time.Sleep(delay)
	}
	ready <- struct{}{}
	return x.WatchExecutions(ctx, client)
}

func (x *OutputExecListener) ReadPastExecutions(ctx context.Context, client *ethclient.Client, contractABI abi.ABI, query ethereum.FilterQuery) error {
	slog.Debug("ReadPastExecutions", "FromBlock", x.FromBlock)

	// Retrieve logs for the specified block range
	oldLogs, err := client.FilterLogs(ctx, query)
	if err != nil {
		return err
	}
	// Process old logs
	for _, vLog := range oldLogs {
		err := x.HandleLog(vLog, client, contractABI)
		if err != nil {
			slog.Error(err.Error())
			continue
		}
	}

	return nil
}

func (x *OutputExecListener) WatchExecutions(ctx context.Context, client *ethclient.Client) error {
	// ABI of your contract
	abi, err := contracts.ApplicationMetaData.GetAbi()
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	contractABI := *abi

	// Subscribe to event
	query := ethereum.FilterQuery{
		FromBlock: x.FromBlock,
		Addresses: []common.Address{x.ApplicationAddress},
		Topics:    [][]common.Hash{{contractABI.Events[x.EventName].ID}},
	}

	var reconnectDelay time.Duration = 5 * time.Second

	for {
		ctxPastInputs, cancel := context.WithCancel(ctx)
		defer cancel()

		err = x.ReadPastExecutions(ctxPastInputs, client, contractABI, query)
		if err != nil {
			slog.Error("unexpected readPastExecutions error", "error", err)
			time.Sleep(reconnectDelay)
			continue
		}

		ctxEth, cancel := context.WithCancel(ctx)
		defer cancel()

		// subscribe to new logs
		logs := make(chan types.Log)
		sub, err := client.SubscribeFilterLogs(ctxEth, query, logs)
		if err != nil {
			slog.Error("unexpected subscribe error", "error", err)
			time.Sleep(reconnectDelay)
			continue
		}

		slog.Info("Listening for execution events...")

		errChannel := make(chan error, 1)

		go func() {
			// Process events
			for {
				select {
				case <-ctxEth.Done():
					errChannel <- ctxEth.Err()
					return
				case err := <-sub.Err():
					errChannel <- err
					return
				case vLog := <-logs:
					if err := x.HandleLog(vLog, client, contractABI); err != nil {
						slog.Error(err.Error())
						// errChannel <- err
						continue
					}
				}
			}
		}()

		err = <-errChannel
		sub.Unsubscribe()

		if ctxEth.Err() != nil {
			return ctxEth.Err()
		}

		if err != nil {
			slog.Error("OutputExecListener", "error", err)
			slog.Info("OutputExecListener reconnecting", "reconnectDelay", reconnectDelay)
			time.Sleep(reconnectDelay)
		} else {
			return nil
		}
	}
}

func (x *OutputExecListener) HandleLog(
	vLog types.Log,
	client *ethclient.Client,
	contractABI abi.ABI,
) error {
	timestamp, blockNumber, values, err := x.GetEventData(
		vLog,
		client,
		contractABI,
	)
	if err != nil {
		return err
	}
	err = x.OnEvent(values, timestamp, blockNumber)
	if err != nil {
		return err
	}
	return nil
}

func (x *OutputExecListener) GetEventData(
	vLog types.Log,
	client *ethclient.Client,
	contractABI abi.ABI,
) (uint64, uint64, []interface{}, error) {
	// Get the block number of the event
	blockNumber := vLog.BlockNumber
	blockNumberBigInt := big.NewInt(int64(blockNumber))
	// Fetch the block information
	block, err := client.BlockByNumber(context.Background(), blockNumberBigInt)
	if err != nil {
		return 0, 0, nil, err
	}

	// Extract the timestamp from the block
	timestamp := block.Time()

	values, err := contractABI.Unpack(x.EventName, vLog.Data)
	if err != nil {
		return 0, 0, nil, err
	}
	return timestamp, blockNumber, values, nil
}
