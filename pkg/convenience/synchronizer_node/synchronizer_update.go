package synchronizernode

import (
	"context"
	"log/slog"

	"github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/cartesi/rollups-graphql/pkg/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
)

const DefaultBatchSize = 50

type SynchronizerUpdate struct {
	DbRawUrl              string
	RawNodeRepository     *RawRepository
	RawInputRefRepository *repository.RawInputRefRepository
	InputRepository       *repository.InputRepository
	BatchSize             int
}

func NewSynchronizerUpdate(
	rawInputRefRepository *repository.RawInputRefRepository,
	rawNode *RawRepository,
	inputRepository *repository.InputRepository,
) SynchronizerUpdate {
	return SynchronizerUpdate{
		RawNodeRepository:     rawNode,
		RawInputRefRepository: rawInputRefRepository,
		BatchSize:             DefaultBatchSize,
		InputRepository:       inputRepository,
	}
}

func (s *SynchronizerUpdate) getFirstRefWithStatusNone(ctx context.Context) (*repository.RawInputRef, error) {
	return s.RawInputRefRepository.FindFirstInputByStatusNone(ctx)
}

func (s *SynchronizerUpdate) findFirst50RawInputsGteRefWithStatus(
	ctx context.Context,
	inputRef repository.RawInputRef,
	status string,
) ([]RawInput, error) {
	return s.RawNodeRepository.First50RawInputsGteRefWithStatus(ctx, inputRef, status)
}

func (s *SynchronizerUpdate) startTransaction(ctx context.Context) (context.Context, error) {
	db := s.RawInputRefRepository.Db
	ctxWithTx, err := repository.StartTransaction(ctx, &db)
	if err != nil {
		return ctx, err
	}
	return ctxWithTx, nil
}

func (s *SynchronizerUpdate) commitTransaction(ctx context.Context) error {
	tx, hasTx := repository.GetTransaction(ctx)
	if hasTx && tx != nil {
		return tx.Commit()
	}
	return nil
}

func (s *SynchronizerUpdate) rollbackTransaction(ctx context.Context) {
	tx, hasTx := repository.GetTransaction(ctx)
	if hasTx && tx != nil {
		err := tx.Rollback()
		if err != nil {
			slog.Error("transaction rollback error", "err", err)
			panic(err)
		}
	}
}

func (s *SynchronizerUpdate) toInputRef(rawInputs []RawInput) []repository.RawInputRef {
	ids := make([]repository.RawInputRef, len(rawInputs))
	for i, input := range rawInputs {
		ids[i] = repository.RawInputRef{
			AppID:      uint64(rawInputs[i].ApplicationId),
			InputIndex: input.Index,
		}
	}
	return ids
}

type RosettaStatusRef struct {
	RawStatus string
	Status    model.CompletionStatus
}

func GetStatusRosetta() []RosettaStatusRef {
	return []RosettaStatusRef{
		{
			RawStatus: "ACCEPTED",
			Status:    model.CompletionStatusAccepted,
		},
		{
			RawStatus: "REJECTED",
			Status:    model.CompletionStatusRejected,
		},
		{
			RawStatus: "EXCEPTION",
			Status:    model.CompletionStatusException,
		},
		{
			RawStatus: "MACHINE_HALTED",
			Status:    model.CompletionStatusMachineHalted,
		},
		{
			RawStatus: "CYCLE_LIMIT_EXCEEDED",
			Status:    model.CompletionStatusCycleLimitExceeded,
		},
		{
			RawStatus: "TIME_LIMIT_EXCEEDED",
			Status:    model.CompletionStatusTimeLimitExceeded,
		},
		{
			RawStatus: "PAYLOAD_LENGTH_LIMIT_EXCEEDED",
			Status:    model.CompletionStatusPayloadLengthLimitExceeded,
		},
	}
}

// if we have a real ID it could be just one sql command using `id in (?)`
func (s *SynchronizerUpdate) updateStatus(ctx context.Context, rawInputs []RawInput, status model.CompletionStatus) error {
	for _, rawInput := range rawInputs {
		appContract := common.BytesToAddress(rawInput.ApplicationAddress)
		// slog.Debug("Update", "appContract", appContract, "index", rawInput.Index, "status", status)
		err := s.InputRepository.UpdateStatus(ctx, appContract, rawInput.Index, status)
		if err != nil {
			slog.Warn("Ignoring missing input", "err", err)
		}
	}
	return nil
}

func (s *SynchronizerUpdate) updateManyInputAndRefsStatus(ctx context.Context, rawInputs []RawInput, rosetta RosettaStatusRef) error {
	err := s.RawInputRefRepository.UpdateStatus(ctx, s.toInputRef(rawInputs), rosetta.RawStatus)
	if err != nil {
		return err
	}
	err = s.updateStatus(ctx, rawInputs, rosetta.Status)
	if err != nil {
		return err
	}
	return nil
}

func (s *SynchronizerUpdate) SyncInputStatus(ctx context.Context) error {
	ctxWithTx, err := s.startTransaction(ctx)
	if err != nil {
		return err
	}
	inputRef, err := s.getFirstRefWithStatusNone(ctxWithTx)
	if err != nil {
		return err
	}
	if inputRef != nil {
		rosettaStone := GetStatusRosetta()
		for _, rosetta := range rosettaStone {
			rawInputs, err := s.findFirst50RawInputsGteRefWithStatus(ctx, *inputRef, rosetta.RawStatus)
			if err != nil {
				s.rollbackTransaction(ctxWithTx)
				return err
			}
			err = s.updateManyInputAndRefsStatus(ctxWithTx, rawInputs, rosetta)
			if err != nil {
				s.rollbackTransaction(ctxWithTx)
				return err
			}
		}
	}
	err = s.commitTransaction(ctxWithTx)
	if err != nil {
		return err
	}
	return nil
}
