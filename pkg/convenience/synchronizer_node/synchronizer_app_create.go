package synchronizernode

import (
	"context"
	"log/slog"

	"github.com/cartesi/rollups-graphql/v2/pkg/convenience/repository"
)

type SynchronizerAppCreator struct {
	AppRepository *repository.ApplicationRepository
	RawRepository *RawRepository
}

func NewSynchronizerAppCreator(
	AppRepository *repository.ApplicationRepository,
	RawRepository *RawRepository,
) *SynchronizerAppCreator {
	return &SynchronizerAppCreator{
		AppRepository,
		RawRepository,
	}
}

func (s *SynchronizerAppCreator) startTransaction(ctx context.Context) (context.Context, error) {
	db := s.AppRepository.Db
	ctxWithTx, err := repository.StartTransaction(ctx, &db)
	if err != nil {
		return ctx, err
	}
	return ctxWithTx, nil
}

func (s *SynchronizerAppCreator) rollbackTransaction(ctx context.Context) {
	tx, hasTx := repository.GetTransaction(ctx)
	if hasTx && tx != nil {
		err := tx.Rollback()
		if err != nil {
			slog.Error("transaction rollback error", "err", err)
			panic(err)
		}
	}
}

func (s *SynchronizerAppCreator) commitTransaction(ctx context.Context) error {
	tx, hasTx := repository.GetTransaction(ctx)
	if hasTx && tx != nil {
		return tx.Commit()
	}
	return nil
}

func (s SynchronizerAppCreator) SyncApps(ctx context.Context) error {
	txCtx, err := s.startTransaction(ctx)
	if err != nil {
		return err
	}
	err = s.syncApps(txCtx)
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

func (s *SynchronizerAppCreator) syncApps(ctx context.Context) error {
	lastAppRef, err := s.AppRepository.GetLatestApp(ctx)
	if err != nil {
		return err
	}
	apps, err := s.RawRepository.GetApplicationRef(ctx, lastAppRef)
	if err != nil {
		return err
	}
	for _, rawApp := range apps {
		app := rawApp.ToConvenience()
		_, err = s.AppRepository.Create(ctx, &app)
		if err != nil {
			return err
		}
	}

	return nil
}
