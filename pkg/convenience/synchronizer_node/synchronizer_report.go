package synchronizernode

import (
	"context"
	"log/slog"

	"github.com/cartesi/rollups-graphql/v2/pkg/convenience/model"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
)

type SynchronizerReport struct {
	ReportRepository *repository.ReportRepository
	RawRepository    *RawRepository
}

func NewSynchronizerReport(
	reportRepository *repository.ReportRepository,
	rawRepository *RawRepository,
) *SynchronizerReport {
	return &SynchronizerReport{
		ReportRepository: reportRepository,
		RawRepository:    rawRepository,
	}
}

func (s *SynchronizerReport) SyncReports(ctx context.Context) error {
	txCtx, err := s.startTransaction(ctx)
	if err != nil {
		return err
	}
	err = s.syncReports(txCtx)
	if err != nil {
		s.rollbackTransaction(txCtx)
		return err
	}
	err = s.commitTransaction(txCtx)
	if err != nil {
		slog.ErrorContext(ctx, "report commit transaction failed")
		panic(err)
	}
	return nil
}

func (s *SynchronizerReport) syncReports(ctx context.Context) error {
	ourLastReport, err := s.ReportRepository.FindLastReport(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "fail to find last report imported")
		return err
	}
	rawReports, err := s.RawRepository.FindAllReportsGt(ctx, ourLastReport)
	if err != nil {
		slog.ErrorContext(ctx, "fail to find all reports")
		return err
	}
	for _, rawReport := range rawReports {
		_, err = s.ReportRepository.CreateReport(ctx, model.Report{
			AppContract: common.BytesToAddress(rawReport.AppContract),
			Index:       int(rawReport.Index),
			InputIndex:  int(rawReport.InputIndex),
			Payload:     common.Bytes2Hex(rawReport.RawData),
			AppID:       rawReport.ApplicationId,
		})
		if err != nil {
			slog.ErrorContext(ctx, "fail to create report", "err", err)
			return err
		}
	}
	return nil
}

func (s *SynchronizerReport) startTransaction(ctx context.Context) (context.Context, error) {
	db := s.ReportRepository.Db
	ctxWithTx, err := repository.StartTransaction(ctx, db)
	if err != nil {
		return ctx, err
	}
	return ctxWithTx, nil
}

func (s *SynchronizerReport) commitTransaction(ctx context.Context) error {
	tx, hasTx := repository.GetTransaction(ctx)
	if hasTx && tx != nil {
		return tx.Commit()
	}
	return nil
}

func (s *SynchronizerReport) rollbackTransaction(ctx context.Context) {
	tx, hasTx := repository.GetTransaction(ctx)
	if hasTx && tx != nil {
		err := tx.Rollback()
		if err != nil {
			slog.ErrorContext(ctx, "transaction rollback error", "err", err)
			panic(err)
		}
	}
}
