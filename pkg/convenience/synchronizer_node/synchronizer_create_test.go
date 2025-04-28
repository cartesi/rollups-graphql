package synchronizernode

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/cartesi/rollups-graphql/v2/pkg/commons"
	"github.com/cartesi/rollups-graphql/v2/pkg/contracts"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience/repository"
	"github.com/cartesi/rollups-graphql/v2/postgres/raw"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

type SynchronizerNodeSuite struct {
	suite.Suite
	ctx                        context.Context
	db                         *sqlx.DB
	dbNodeV2                   *sqlx.DB
	dockerComposeStartedByTest bool
	workerCtx                  context.Context
	timeoutCancel              context.CancelFunc
	workerCancel               context.CancelFunc
	workerResult               chan error
	inputRepository            *repository.InputRepository
	inputRefRepository         *repository.RawInputRefRepository
	dbFactory                  *commons.DbFactory
}

func (s *SynchronizerNodeSuite) SetupSuite() {
	timeout := 1 * time.Minute
	s.ctx, s.timeoutCancel = context.WithTimeout(context.Background(), timeout)

	pgUp := commons.IsPortInUse(5432)
	if !pgUp {
		err := raw.RunDockerCompose(s.ctx)
		s.NoError(err)
		s.dockerComposeStartedByTest = true
	}
}

func (s *SynchronizerNodeSuite) SetupTest() {
	var err error
	commons.ConfigureLog(slog.LevelDebug)

	s.workerResult = make(chan error)

	// Database
	s.dbFactory, err = commons.NewDbFactory()
	s.Require().NoError(err)
	s.db, err = s.dbFactory.CreateDbCtx(s.ctx, "input.sqlite3")
	s.Require().NoError(err)
	container := convenience.NewContainer(s.db, false)
	s.inputRepository = container.GetInputRepository(s.ctx)
	s.inputRefRepository = &repository.RawInputRefRepository{Db: s.db}
	err = s.inputRefRepository.CreateTables(s.ctx)
	s.NoError(err)

	s.workerCtx, s.workerCancel = context.WithCancel(s.ctx)

	s.dbNodeV2 = sqlx.MustConnect("postgres", RAW_DB_URL)
	rawRepository := RawRepository{Db: s.dbNodeV2}
	synchronizerUpdate := NewSynchronizerUpdate(
		s.inputRefRepository,
		&rawRepository,
		s.inputRepository,
	)
	synchronizerReport := NewSynchronizerReport(
		container.GetReportRepository(s.ctx),
		&rawRepository,
	)
	synchronizerOutputUpdate := NewSynchronizerOutputUpdate(
		container.GetVoucherRepository(s.ctx),
		container.GetNoticeRepository(s.ctx),
		&rawRepository,
		container.GetRawOutputRefRepository(s.ctx),
	)

	abi, err := contracts.OutputsMetaData.GetAbi()
	s.Require().NoError(err)

	abiDecoder := NewAbiDecoder(abi)

	synchronizerOutputCreate := NewSynchronizerOutputCreate(
		container.GetVoucherRepository(s.ctx),
		container.GetNoticeRepository(s.ctx),
		&rawRepository,
		container.GetRawOutputRefRepository(s.ctx),
		abiDecoder,
	)

	synchronizerOutputExecuted := NewSynchronizerOutputExecuted(
		container.GetVoucherRepository(s.ctx),
		container.GetNoticeRepository(s.ctx),
		&rawRepository,
		container.GetRawOutputRefRepository(s.ctx),
	)

	synchronizerCreateInput := NewSynchronizerInputCreator(
		container.GetInputRepository(s.ctx),
		container.GetRawInputRepository(s.ctx),
		&rawRepository,
		abiDecoder,
	)

	synchronizerAppCreate := NewSynchronizerAppCreator(container.GetApplicationRepository(s.ctx), &rawRepository)

	wr := NewSynchronizerCreateWorker(
		s.inputRepository,
		s.inputRefRepository,
		RAW_DB_URL,
		&rawRepository,
		&synchronizerUpdate,
		container.GetOutputDecoder(s.ctx),
		synchronizerAppCreate,
		synchronizerReport,
		synchronizerOutputUpdate,
		container.GetRawOutputRefRepository(s.ctx),
		synchronizerOutputCreate,
		synchronizerCreateInput,
		synchronizerOutputExecuted,
	)

	// like Supervisor
	ready := make(chan struct{})
	go func() {
		s.workerResult <- wr.Start(s.workerCtx, ready)
	}()
	select {
	case <-s.ctx.Done():
		s.Fail("context error", s.ctx.Err())
	case err := <-s.workerResult:
		s.Fail("worker exited before being ready", err)
	case <-ready:
		s.T().Log("worker ready")
	}
}

func (s *SynchronizerNodeSuite) TearDownSuite() {
	if s.dockerComposeStartedByTest {
		err := raw.StopDockerCompose(s.ctx)
		s.NoError(err)
	}
	s.timeoutCancel()
}

func (s *SynchronizerNodeSuite) TearDownTest() {
	time.Sleep(1 * time.Second) // wait for io
	err := s.db.Close()
	s.NoError(err)
	err = s.dbNodeV2.Close()
	s.NoError(err)
	s.dbFactory.Cleanup(s.ctx)
	s.workerCancel()
}

func TestSynchronizerNodeSuite(t *testing.T) {
	suite.Run(t, new(SynchronizerNodeSuite))
}

func (s *SynchronizerNodeSuite) XTestSynchronizerNodeConnection() {
	val := <-s.workerResult
	s.NoError(val)
}

func (s *SynchronizerNodeSuite) TestFormatTransactionId() {
	data := []byte{1, 1}
	id := FormatTransactionId(data)
	s.Equal("257", id)

	data = []byte{17}
	id = FormatTransactionId(data)
	s.Equal("17", id)

	data = crypto.Keccak256([]byte(data))
	id = FormatTransactionId(data)
	s.Equal("0x0552ab8dc52e1cf9328ddb97e0966b9c88de9cca97f48b0110d7800982596158", id)
}

func (s *SynchronizerNodeSuite) TestFormatTransactionIdAlwaysA32Bytes() {
	data := common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000")
	id := FormatTransactionId(data)
	s.Equal("0", id)

	data = common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")
	id = FormatTransactionId(data)
	s.Equal("1", id)

	data = common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000a")
	id = FormatTransactionId(data)
	s.Equal("10", id)

	data = common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000002a")
	id = FormatTransactionId(data)
	s.Equal("42", id)

	data = []byte{17}
	id = FormatTransactionId(data)
	s.Equal("17", id)

	data = crypto.Keccak256([]byte(data))
	id = FormatTransactionId(data)
	s.Equal("0x0552ab8dc52e1cf9328ddb97e0966b9c88de9cca97f48b0110d7800982596158", id)
}
