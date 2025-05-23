package synchronizernode

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"

	"github.com/cartesi/rollups-graphql/v2/pkg/convenience/model"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
)

type SynchronizerOutputCreate struct {
	VoucherRepository      *repository.VoucherRepository
	NoticeRepository       *repository.NoticeRepository
	RawNodeV2Repository    *RawRepository
	RawOutputRefRepository *repository.RawOutputRefRepository
	AbiDecoder             *AbiDecoder
}

func NewSynchronizerOutputCreate(
	voucherRepository *repository.VoucherRepository,
	noticeRepository *repository.NoticeRepository,
	rawRepository *RawRepository,
	rawOutputRefRepository *repository.RawOutputRefRepository,
	abiDecoder *AbiDecoder,
) *SynchronizerOutputCreate {
	return &SynchronizerOutputCreate{
		VoucherRepository:      voucherRepository,
		NoticeRepository:       noticeRepository,
		RawNodeV2Repository:    rawRepository,
		RawOutputRefRepository: rawOutputRefRepository,
		AbiDecoder:             abiDecoder,
	}
}

func (s *SynchronizerOutputCreate) SyncOutputs(ctx context.Context) error {
	txCtx, err := s.startTransaction(ctx)
	if err != nil {
		return err
	}
	err = s.syncOutputs(txCtx)
	if err != nil {
		s.rollbackTransaction(txCtx)
		return err
	}
	err = s.commitTransaction(txCtx)
	if err != nil {
		return err
	}
	return nil
}

func (s *SynchronizerOutputCreate) syncOutputs(ctx context.Context) error {
	latestOutputRef, err := s.RawOutputRefRepository.FindLatestRawOutputRef(ctx)
	if err != nil {
		return err
	}
	if latestOutputRef != nil {
		slog.DebugContext(ctx, "SyncOutputs",
			"CreatedAt", latestOutputRef.CreatedAt,
			"AppID", latestOutputRef.AppID,
			"OutputIndex", latestOutputRef.OutputIndex,
		)
	}
	outputs, err := s.RawNodeV2Repository.FindAllOutputsGtRefLimited(ctx, latestOutputRef)
	if err != nil {
		return err
	}
	for _, rawOutput := range outputs {
		rawOutputRef, err := s.ToRawOutputRef(rawOutput)
		if err != nil {
			return err
		}
		err = s.RawOutputRefRepository.Create(ctx, *rawOutputRef)
		if err != nil {
			return err
		}
		err = s.CreateOutput(ctx, rawOutputRef, rawOutput)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SynchronizerOutputCreate) CreateOutput(ctx context.Context, rawOutputRef *repository.RawOutputRef, rawOutput Output) error {
	if rawOutputRef.Type == repository.RAW_VOUCHER_TYPE {
		cVoucher, err := s.ToConvenienceVoucher(rawOutput)
		if err != nil {
			return err
		}
		_, err = s.VoucherRepository.CreateVoucher(ctx, cVoucher)
		if err != nil {
			return err
		}
	} else if rawOutputRef.Type == repository.RAW_NOTICE_TYPE {
		cNotice, err := s.ToConvenienceNotice(rawOutput)
		if err != nil {
			return err
		}
		_, err = s.NoticeRepository.Create(ctx, cNotice)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unexpected output type")
	}
	return nil
}

func (s *SynchronizerOutputCreate) ToConvenienceVoucher(rawOutput Output) (*model.ConvenienceVoucher, error) {
	data, err := s.AbiDecoder.GetMapRaw(rawOutput.RawData)
	if err != nil {
		return nil, err
	}
	destination, ok := data["destination"].(common.Address)
	if !ok {
		return nil, fmt.Errorf("destination not found %v", data)
	}

	voucherValue, ok := data["value"].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("value not found %v", data)
	}
	strPayload := "0x" + common.Bytes2Hex(rawOutput.RawData)
	cVoucher := model.ConvenienceVoucher{
		Destination:      destination,
		Payload:          strPayload,
		Executed:         false,
		InputIndex:       rawOutput.InputIndex,
		OutputIndex:      rawOutput.Index,
		ProofOutputIndex: rawOutput.Index,
		AppContract:      common.BytesToAddress(rawOutput.AppContract),
		Value:            voucherValue.String(),
	}
	return &cVoucher, nil
}

func (s *SynchronizerOutputCreate) ToConvenienceNotice(rawOutput Output) (*model.ConvenienceNotice, error) {
	strPayload := "0x" + common.Bytes2Hex(rawOutput.RawData)
	cNotice := model.ConvenienceNotice{
		Payload:              strPayload,
		InputIndex:           rawOutput.InputIndex,
		OutputIndex:          rawOutput.Index,
		ProofOutputIndex:     rawOutput.Index,
		AppContract:          common.BytesToAddress(rawOutput.AppContract).Hex(),
		OutputHashesSiblings: string(rawOutput.OutputHashesSiblings),
	}
	return &cNotice, nil
}

func (s *SynchronizerOutputCreate) ToRawOutputRef(rawOutput Output) (*repository.RawOutputRef, error) {
	outputType, err := getOutputType(rawOutput.RawData)
	if err != nil {
		return nil, err
	}
	return &repository.RawOutputRef{
		AppID:       rawOutput.ApplicationId,
		InputIndex:  rawOutput.InputIndex,
		OutputIndex: rawOutput.Index,
		AppContract: common.BytesToAddress(rawOutput.AppContract).Hex(),
		Type:        outputType,
		UpdatedAt:   rawOutput.UpdatedAt,
	}, nil
}

func getOutputType(rawData []byte) (string, error) {
	var strPayload = "0x" + common.Bytes2Hex(rawData)
	if strPayload[2:10] == model.VOUCHER_SELECTOR {
		return repository.RAW_VOUCHER_TYPE, nil
	} else if strPayload[2:10] == model.NOTICE_SELECTOR {
		return repository.RAW_NOTICE_TYPE, nil
	} else {
		return "", fmt.Errorf("unsupported output selector type: %s", strPayload[2:10])
	}
}

func (s *SynchronizerOutputCreate) startTransaction(ctx context.Context) (context.Context, error) {
	db := s.RawOutputRefRepository.Db
	ctxWithTx, err := repository.StartTransaction(ctx, db)
	if err != nil {
		return ctx, err
	}
	return ctxWithTx, nil
}

func (s *SynchronizerOutputCreate) commitTransaction(ctx context.Context) error {
	tx, hasTx := repository.GetTransaction(ctx)
	if hasTx && tx != nil {
		return tx.Commit()
	}
	return nil
}

func (s *SynchronizerOutputCreate) rollbackTransaction(ctx context.Context) {
	tx, hasTx := repository.GetTransaction(ctx)
	if hasTx && tx != nil {
		err := tx.Rollback()
		if err != nil {
			slog.ErrorContext(ctx, "transaction rollback error", "err", err)
			panic(err)
		}
	}
}
