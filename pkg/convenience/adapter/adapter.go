package adapter

import (
	"fmt"
	"log/slog"

	"github.com/cartesi/rollups-graphql/pkg/contracts"
	"github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/ethereum/go-ethereum/common"
)

func ConvertVoucherPayloadToV2(payloadV1 string) string {
	return fmt.Sprintf("0x%s%s", model.VOUCHER_SELECTOR, payloadV1)
}

func ConvertNoticePayloadToV2(payloadV1 string) string {
	return fmt.Sprintf("0x%s%s", model.NOTICE_SELECTOR, payloadV1)
}

// for a while we will remove the prefix
// until the v2 does not arrives
func RemoveSelector(payload string) string {
	return fmt.Sprintf("0x%s", payload[10:])
}

func GetDestination(payload string) (common.Address, error) {
	abiParsed, err := contracts.OutputsMetaData.GetAbi()

	if err != nil {
		slog.Error("Error parsing abi", "err", err)
		return common.Address{}, err
	}

	slog.Info("payload", "payload", payload)

	values, err := abiParsed.Methods["Voucher"].Inputs.Unpack(common.Hex2Bytes(payload[10:]))

	if err != nil {
		slog.Error("Error unpacking abi", "err", err)
		return common.Address{}, err
	}

	return values[0].(common.Address), nil
}

func GetConvertedInput(payload string) ([]interface{}, error) {
	abiParsed, err := contracts.InputsMetaData.GetAbi()

	if err != nil {
		slog.Error("Error parsing abi", "err", err)
		return make([]interface{}, 0), err
	}

	values, err := abiParsed.Methods["EvmAdvance"].Inputs.Unpack(common.Hex2Bytes(payload[10:]))

	if err != nil {
		slog.Error("Error unpacking abi", "err", err)
		return make([]interface{}, 0), err
	}

	return values, nil

}
