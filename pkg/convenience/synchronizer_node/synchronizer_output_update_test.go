package synchronizernode

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/cartesi/rollups-graphql/v2/pkg/commons"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience/model"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience/repository"
	"github.com/cartesi/rollups-graphql/v2/postgres/raw"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

type SynchronizerOutputUpdateSuite struct {
	suite.Suite
	ctx                        context.Context
	dockerComposeStartedByTest bool
	tempDir                    string
	container                  *convenience.Container
	synchronizerOutputUpdate   *SynchronizerOutputUpdate
	rawNodeV2Repository        *RawRepository
}

func (s *SynchronizerOutputUpdateSuite) SetupSuite() {
	pgUp := commons.IsPortInUse(5432)
	if !pgUp {
		err := raw.RunDockerCompose(s.ctx)
		s.NoError(err)
		s.dockerComposeStartedByTest = true
	}
}

func (s *SynchronizerOutputUpdateSuite) SetupTest() {
	s.ctx = context.Background()
	commons.ConfigureLog(slog.LevelDebug)

	// Temp
	tempDir, err := os.MkdirTemp("", "")
	s.NoError(err)
	s.tempDir = tempDir

	// Database
	sqliteFileName := filepath.Join(tempDir, "output.sqlite3")
	// sqliteFileName = fmt.Sprintf("../../../sync-proof-output-%d.sqlite3", time.Now().Unix())

	db := sqlx.MustConnect("sqlite3", sqliteFileName)
	s.container = convenience.NewContainer(db, false)

	dbNodeV2 := sqlx.MustConnect("postgres", RAW_DB_URL)
	s.rawNodeV2Repository = NewRawRepository(RAW_DB_URL, dbNodeV2)

	s.synchronizerOutputUpdate = NewSynchronizerOutputUpdate(
		s.container.GetVoucherRepository(s.ctx),
		s.container.GetNoticeRepository(s.ctx),
		s.rawNodeV2Repository,
		s.container.GetRawOutputRefRepository(s.ctx),
	)
}

func (s *SynchronizerOutputUpdateSuite) TearDownSuite() {
	if s.dockerComposeStartedByTest {
		err := raw.StopDockerCompose(s.ctx)
		s.NoError(err)
	}
}

func (s *SynchronizerOutputUpdateSuite) TearDownTest() {
	defer os.RemoveAll(s.tempDir)
}

func TestSynchronizerOutputUpdateSuiteSuite(t *testing.T) {
	suite.Run(t, new(SynchronizerOutputUpdateSuite))
}

// Dear Programmer, I hope this message finds you well.
// Keep coding, keep learning, and never forget—your work shapes the future.
func (s *SynchronizerOutputUpdateSuite) TestUpdateOutputsProofs() {
	ctx := context.Background()

	s.fillRefData(ctx)

	// check setup
	proofCount := s.countHLProofs(ctx)
	s.Require().Equal(0, proofCount)

	// first call
	err := s.synchronizerOutputUpdate.SyncOutputsProofs(ctx)
	s.Require().NoError(err)
	first := s.countHLProofs(ctx)
	s.Equal((TOTAL_INPUT_TEST / 2), first)

	// second call
	err = s.synchronizerOutputUpdate.SyncOutputsProofs(ctx)
	s.Require().NoError(err)
	second := s.countHLProofs(ctx)
	s.Equal(TOTAL_INPUT_TEST, second)

	// third call
	err = s.synchronizerOutputUpdate.SyncOutputsProofs(ctx)
	s.Require().NoError(err)
	third := s.countHLProofs(ctx)
	s.Equal(TOTAL_INPUT_TEST+(TOTAL_INPUT_TEST/2), third)
}

func (s *SynchronizerOutputUpdateSuite) countHLProofs(ctx context.Context) int {
	total, err := s.container.GetOutputRepository().CountProofs(ctx)
	s.Require().NoError(err)
	return int(total)
}

func (s *SynchronizerOutputUpdateSuite) fillRefData(ctx context.Context) {
	txCtx, err := s.synchronizerOutputUpdate.startTransaction(ctx)
	s.Require().NoError(err)
	appContract := common.HexToAddress(DEFAULT_TEST_APP_CONTRACT)
	for i := 0; i < TOTAL_INPUT_TEST*2; i++ {
		// id := strconv.FormatInt(int64(i), 10) // our ID
		outputType := repository.RAW_VOUCHER_TYPE
		if i%2 == 0 {
			outputType = "notice"
		}
		err := s.container.GetRawOutputRefRepository(s.ctx).Create(txCtx, repository.RawOutputRef{
			AppID:       uint64(1),
			InputIndex:  uint64(i),
			OutputIndex: uint64(i),
			AppContract: appContract.Hex(),
			Type:        outputType,
			HasProof:    false,
		})
		s.Require().NoError(err)
		if outputType == repository.RAW_VOUCHER_TYPE {
			_, err = s.container.GetVoucherRepository(s.ctx).CreateVoucher(
				txCtx, &model.ConvenienceVoucher{
					AppContract: appContract,
					OutputIndex: uint64(i),
					InputIndex:  uint64(i),
				},
			)
			s.Require().NoError(err)
		} else {
			_, err = s.container.GetNoticeRepository(s.ctx).Create(
				txCtx, &model.ConvenienceNotice{
					AppContract: appContract.Hex(),
					OutputIndex: uint64(i),
					InputIndex:  uint64(i),
				},
			)
			s.Require().NoError(err)
		}
	}
	err = s.synchronizerOutputUpdate.commitTransaction(txCtx)
	s.Require().NoError(err)
}
