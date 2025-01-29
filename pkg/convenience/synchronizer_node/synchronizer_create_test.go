package synchronizernode

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/cartesi/rollups-graphql/pkg/contracts"
	"github.com/cartesi/rollups-graphql/pkg/convenience"
	"github.com/cartesi/rollups-graphql/pkg/convenience/repository"
	"github.com/cartesi/rollups-graphql/postgres/raw"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

type SynchronizerNodeSuite struct {
	suite.Suite
	ctx                        context.Context
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
	commons.ConfigureLog(slog.LevelDebug)

	s.workerResult = make(chan error)

	// Database
	s.dbFactory = commons.NewDbFactory()
	db, err := s.dbFactory.CreateDbCtx(s.ctx, "input.sqlite3")
	s.NoError(err)
	container := convenience.NewContainer(*db, false)
	s.inputRepository = container.GetInputRepository()
	s.inputRefRepository = &repository.RawInputRefRepository{Db: *db}
	err = s.inputRefRepository.CreateTables()
	s.NoError(err)

	s.workerCtx, s.workerCancel = context.WithCancel(s.ctx)

	dbNodeV2 := sqlx.MustConnect("postgres", RAW_DB_URL)
	rawRepository := RawRepository{Db: dbNodeV2}
	synchronizerUpdate := NewSynchronizerUpdate(
		s.inputRefRepository,
		&rawRepository,
		s.inputRepository,
	)
	synchronizerReport := NewSynchronizerReport(
		container.GetReportRepository(),
		&rawRepository,
	)
	synchronizerOutputUpdate := NewSynchronizerOutputUpdate(
		container.GetVoucherRepository(),
		container.GetNoticeRepository(),
		&rawRepository,
		container.GetRawOutputRefRepository(),
	)

	abi, err := contracts.OutputsMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	abiDecoder := NewAbiDecoder(abi)

	synchronizerOutputCreate := NewSynchronizerOutputCreate(
		container.GetVoucherRepository(),
		container.GetNoticeRepository(),
		&rawRepository,
		container.GetRawOutputRefRepository(),
		abiDecoder,
	)

	synchronizerOutputExecuted := NewSynchronizerOutputExecuted(
		container.GetVoucherRepository(),
		container.GetNoticeRepository(),
		&rawRepository,
		container.GetRawOutputRefRepository(),
	)

	synchronizerCreateInput := NewSynchronizerInputCreator(
		container.GetInputRepository(),
		container.GetRawInputRepository(),
		&rawRepository,
		abiDecoder,
	)
	wr := NewSynchronizerCreateWorker(
		s.inputRepository,
		s.inputRefRepository,
		RAW_DB_URL,
		&rawRepository,
		&synchronizerUpdate,
		container.GetOutputDecoder(),
		synchronizerReport,
		synchronizerOutputUpdate,
		container.GetRawOutputRefRepository(),
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
	s.dbFactory.Cleanup()
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
