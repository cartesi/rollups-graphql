package convenience

import (
	"log/slog"
	"testing"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/cartesi/rollups-graphql/pkg/convenience/repository"
	"github.com/cartesi/rollups-graphql/pkg/convenience/services"
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
}

var Bob = common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
var Bruno = common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC")
var Alice = common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
var Token = common.HexToAddress("0xc6e7DF5E7b4f2A278906862b61205850344D4e7d")

func (s *ExecListenerSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	db := sqlx.MustConnect("sqlite3", ":memory:")
	s.repository = &repository.VoucherRepository{
		Db: *db,
	}
	err := s.repository.CreateTables()
	if err != nil {
		panic(err)
	}

	s.noticeRepository = &repository.NoticeRepository{
		Db: *db,
	}
	err = s.noticeRepository.CreateTables()
	if err != nil {
		panic(err)
	}

	s.inputRepository = &repository.InputRepository{
		Db: *db,
	}
	err = s.inputRepository.CreateTables()
	if err != nil {
		panic(err)
	}

	s.reportRepository = &repository.ReportRepository{
		Db: db,
	}

	s.applicationRepository = &repository.ApplicationRepository{
		Db: *db,
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
