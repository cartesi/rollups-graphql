package repository

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/cartesi/rollups-graphql/v2/pkg/commons"
	configtest "github.com/cartesi/rollups-graphql/v2/pkg/convenience/config_test"
	"github.com/stretchr/testify/suite"
)

type RawOutputRefSuite struct {
	suite.Suite
	noticeRepository       *NoticeRepository
	rawOutputRefRepository *RawOutputRefRepository
	dbFactory              *commons.DbFactory
	ctx                    context.Context
	ctxCancel              context.CancelFunc
}

func (s *RawOutputRefSuite) TearDownTest() {
	s.dbFactory.Cleanup(s.ctx)
	s.ctxCancel()
}

func TestRawRefOutputSuite(t *testing.T) {
	suite.Run(t, new(RawOutputRefSuite))
}

func (s *RawOutputRefSuite) SetupTest() {
	var err error
	s.ctx, s.ctxCancel = context.WithCancel(context.Background())
	commons.ConfigureLog(slog.LevelDebug)
	s.dbFactory, err = commons.NewDbFactory()
	s.Require().NoError(err)
	db := s.dbFactory.CreateDb(s.ctx, "input.sqlite3")
	s.noticeRepository = &NoticeRepository{
		Db: db,
	}
	s.rawOutputRefRepository = &RawOutputRefRepository{
		Db: db,
	}

	err = s.noticeRepository.CreateTables(s.ctx)
	s.NoError(err)
	err = s.rawOutputRefRepository.CreateTable(s.ctx)
	s.NoError(err)
}

func (s *RawOutputRefSuite) TestRawRefOutputCreateTables() {
	err := s.rawOutputRefRepository.CreateTable(s.ctx)
	s.NoError(err)
}

func (s *RawOutputRefSuite) TestRawRefOutputShouldThrowAnErrorWhenThereIsNoTypeAttribute() {
	rawNotice := RawOutputRef{
		InputIndex:  1,
		AppID:       2,
		AppContract: "0x123456789abcdef",
		OutputIndex: 2,
	}

	err := s.rawOutputRefRepository.Create(s.ctx, rawNotice)
	s.ErrorContains(err, "sqlite3: constraint failed: CHECK constraint failed: type IN ('voucher', 'notice')")
}

func (s *RawOutputRefSuite) TestRawRefOutputShouldThrowAnErrorWhenTypeAttributeIsDiffFromVoucherOrNotice() {
	ctx := context.Background()

	rawNotice := RawOutputRef{
		InputIndex:  1,
		AppID:       2,
		AppContract: "0x123456789abcdef",
		OutputIndex: 2,
		Type:        "report",
	}

	err := s.rawOutputRefRepository.Create(ctx, rawNotice)
	s.ErrorContains(err, "sqlite3: constraint failed: CHECK constraint failed: type IN ('voucher', 'notice')")
}

func (s *RawOutputRefSuite) TestRawRefOutputCreate() {
	ctx := context.Background()

	createdAt := time.Now()
	rawOutputRef := RawOutputRef{
		InputIndex:  1,
		AppID:       2,
		AppContract: configtest.DEFAULT_TEST_APP_CONTRACT,
		OutputIndex: 2,
		Type:        RAW_NOTICE_TYPE,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}

	err := s.rawOutputRefRepository.Create(ctx, rawOutputRef)
	s.NoError(err)

	var count int
	err = s.rawOutputRefRepository.Db.QueryRow(`SELECT COUNT(*) FROM convenience_output_raw_references WHERE input_index = ? AND app_contract = ? AND output_index = ?`,
		rawOutputRef.InputIndex, rawOutputRef.AppContract, rawOutputRef.OutputIndex).Scan(&count)

	s.NoError(err)
	s.Equal(1, count)

	saved, err := s.rawOutputRefRepository.FindByAppIDAndOutputIndex(ctx, 2, 2)
	s.Require().NoError(err)

	// here we have a round problem using sqlite
	s.Equal(createdAt.UnixMilli(), saved.CreatedAt.UnixMilli())
}

func (s *RawOutputRefSuite) TestFindLatestRawOutputRef() {
	ctx := context.Background()

	firstRawOutput := RawOutputRef{
		AppID:       1,
		InputIndex:  1,
		AppContract: "0x123456789abcdef",
		OutputIndex: 2,
		Type:        "notice",
	}

	err := s.rawOutputRefRepository.Create(ctx, firstRawOutput)
	s.NoError(err)

	lastRawOutput := RawOutputRef{
		AppID:       2,
		InputIndex:  2,
		AppContract: "0x123456789abcdef",
		OutputIndex: 23,
		Type:        "voucher",
	}

	err = s.rawOutputRefRepository.Create(ctx, lastRawOutput)
	s.NoError(err)

	var count int
	err = s.rawOutputRefRepository.Db.QueryRow(`SELECT COUNT(*) FROM convenience_output_raw_references`).Scan(&count)
	s.NoError(err)
	//check if there are two records in the table.
	s.Require().Equal(2, count)

	lastOutputRef, err := s.rawOutputRefRepository.FindLatestRawOutputRef(ctx)
	s.NoError(err)
	s.Equal(lastRawOutput.AppID, lastOutputRef.AppID)
}
