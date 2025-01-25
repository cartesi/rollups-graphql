package synchronizernode

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/cartesi/rollups-graphql/pkg/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
)

type SynchronizerOutputExecuted struct {
	VoucherRepository      *repository.VoucherRepository
	NoticeRepository       *repository.NoticeRepository
	RawNodeV2Repository    *RawRepository
	RawOutputRefRepository *repository.RawOutputRefRepository
}

func NewSynchronizerOutputExecuted(
	voucherRepository *repository.VoucherRepository,
	noticeRepository *repository.NoticeRepository,
	rawRepository *RawRepository,
	rawOutputRefRepository *repository.RawOutputRefRepository,
) *SynchronizerOutputExecuted {
	return &SynchronizerOutputExecuted{
		VoucherRepository:      voucherRepository,
		NoticeRepository:       noticeRepository,
		RawNodeV2Repository:    rawRepository,
		RawOutputRefRepository: rawOutputRefRepository,
	}
}

func (s *SynchronizerOutputExecuted) SyncOutputsExecution(ctx context.Context) error {
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

func (s *SynchronizerOutputExecuted) syncOutputs(ctx context.Context) error {
	lastOutputRef, err := s.RawOutputRefRepository.GetLastUpdatedAtExecuted(ctx)
	if err != nil {
		return err
	}
	if lastOutputRef == nil {
		lastOutputRef = &repository.RawOutputRef{
			AppID:       0,
			OutputIndex: 0,
			InputIndex:  0,
			UpdatedAt:   time.Unix(0, 0),
		}
	} else {
		slog.Debug("SyncOutputs",
			"lastUpdatedAtExecuted", lastOutputRef.UpdatedAt,
			"AppID", lastOutputRef.AppID,
			"OutputIndex", lastOutputRef.OutputIndex,
		)
	}

	rawOutputs, err := s.RawNodeV2Repository.FindAllOutputsExecutedAfter(ctx, lastOutputRef)
	if err != nil {
		return err
	}
	for _, rawOutput := range rawOutputs {
		err = s.UpdateExecutionData(ctx, rawOutput)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SynchronizerOutputExecuted) UpdateExecutionData(
	ctx context.Context,
	rawOutput Output,
) error {
	ref, err := s.RawOutputRefRepository.FindByAppIDAndOutputIndex(ctx, rawOutput.ApplicationId, rawOutput.Index)
	if err != nil {
		return err
	}
	if ref == nil {
		slog.Warn("We may need to wait for the reference to be created")
		return nil
	}
	appContract := common.BytesToAddress(rawOutput.AppContract)
	if ref.Type == repository.RAW_VOUCHER_TYPE {
		err = s.VoucherRepository.SetExecuted(ctx,
			&model.ConvenienceVoucher{
				AppContract:     appContract,
				OutputIndex:     rawOutput.Index,
				TransactionHash: "0x" + common.Bytes2Hex(rawOutput.TransactionHash),
			})
		if err != nil {
			return err
		}
	} else if ref.Type == repository.RAW_NOTICE_TYPE {
		slog.Warn("Ignoring executed status because the output is a notice",
			"inputIndex", rawOutput.InputIndex,
			"index", rawOutput.Index,
			"appContract", appContract.Hex(),
		)
	} else {
		return fmt.Errorf("unexpected output type: %s", ref.Type)
	}
	ref.UpdatedAt = rawOutput.UpdatedAt
	err = s.RawOutputRefRepository.SetExecutedToTrue(ctx, ref)
	if err != nil {
		return err
	}
	return nil
}

func (s *SynchronizerOutputExecuted) startTransaction(ctx context.Context) (context.Context, error) {
	db := s.RawOutputRefRepository.Db
	ctxWithTx, err := repository.StartTransaction(ctx, db)
	if err != nil {
		return ctx, err
	}
	return ctxWithTx, nil
}

func (s *SynchronizerOutputExecuted) commitTransaction(ctx context.Context) error {
	tx, hasTx := repository.GetTransaction(ctx)
	if hasTx && tx != nil {
		return tx.Commit()
	}
	return nil
}

func (s *SynchronizerOutputExecuted) rollbackTransaction(ctx context.Context) {
	tx, hasTx := repository.GetTransaction(ctx)
	if hasTx && tx != nil {
		err := tx.Rollback()
		if err != nil {
			slog.Error("transaction rollback error", "err", err)
			panic(err)
		}
	}
}
