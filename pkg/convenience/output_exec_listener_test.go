package convenience

import (
	"context"
	"log/slog"
	"testing"

	"github.com/cartesi/rollups-graphql/v2/pkg/commons"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience/repository"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience/services"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

func TestExecListenerSuite(t *testing.T) {
	suite.Run(t, new(ExecListenerSuite))
}

type ExecListenerSuite struct {
	suite.Suite
	ConvenienceService    *services.ConvenienceService
	repository            *repository.VoucherRepository
	noticeRepository      *repository.NoticeRepository
	inputRepository       *repository.InputRepository
	reportRepository      *repository.ReportRepository
	applicationRepository *repository.ApplicationRepository
	db                    *sqlx.DB
	ctx                   context.Context
	ctxCancel             context.CancelFunc
}

var Bob = common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
var Bruno = common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC")
var Alice = common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
var Token = common.HexToAddress("0xc6e7DF5E7b4f2A278906862b61205850344D4e7d")

func (s *ExecListenerSuite) TearDownTest() {
	s.ctxCancel()
	err := s.db.Close()
	s.NoError(err)
}

func (s *ExecListenerSuite) SetupTest() {
	s.ctx, s.ctxCancel = context.WithCancel(context.Background())
	commons.ConfigureLog(slog.LevelDebug)
	s.db = sqlx.MustConnect("sqlite3", ":memory:")
	s.repository = &repository.VoucherRepository{
		Db: s.db,
	}
	err := s.repository.CreateTables(s.ctx)
	s.Require().NoError(err)

	s.noticeRepository = &repository.NoticeRepository{
		Db: s.db,
	}
	err = s.noticeRepository.CreateTables(s.ctx)
	s.Require().NoError(err)

	s.inputRepository = &repository.InputRepository{
		Db: s.db,
	}
	err = s.inputRepository.CreateTables(s.ctx)
	s.Require().NoError(err)

	s.reportRepository = &repository.ReportRepository{
		Db: s.db,
	}

	s.applicationRepository = &repository.ApplicationRepository{
		Db: s.db,
	}

	s.ConvenienceService = services.NewConvenienceService(
		s.repository,
		s.noticeRepository,
		s.inputRepository,
		s.reportRepository,
		s.applicationRepository,
	)
}

func (s *ExecListenerSuite) TestItUpdateExecutedAtAndBlocknumber() {
	// deprecated
}
