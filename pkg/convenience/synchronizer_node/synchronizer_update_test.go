package synchronizernode

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
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

// Account that sends the transactions.
const SenderAddress = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

type SynchronizerUpdateNodeSuite struct {
	suite.Suite
	ctx                        context.Context
	dockerComposeStartedByTest bool
	tempDir                    string
	container                  *convenience.Container
	synchronizerUpdate         SynchronizerUpdate
	rawNode                    *RawRepository
}

func (s *SynchronizerUpdateNodeSuite) SetupSuite() {
	pgUp := commons.IsPortInUse(5432)
	if !pgUp {
		err := raw.RunDockerCompose(s.ctx)
		s.NoError(err)
		s.dockerComposeStartedByTest = true
	}
}

func (s *SynchronizerUpdateNodeSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)

	// Temp
	tempDir, err := os.MkdirTemp("", "")
	s.NoError(err)
	s.tempDir = tempDir

	// Database
	sqliteFileName := filepath.Join(tempDir, "update_input.sqlite3")
	slog.Debug("SetupTest", "sqliteFileName", sqliteFileName)
	db := sqlx.MustConnect("sqlite3", sqliteFileName)
	s.container = convenience.NewContainer(*db, false)

	dbNodeV2 := sqlx.MustConnect("postgres", RAW_DB_URL)
	s.rawNode = NewRawRepository(RAW_DB_URL, dbNodeV2)
	rawInputRefRepository := s.container.GetRawInputRepository()
	s.synchronizerUpdate = NewSynchronizerUpdate(
		rawInputRefRepository,
		s.rawNode,
		s.container.GetInputRepository(),
	)
}

func (s *SynchronizerUpdateNodeSuite) TearDownSuite() {
	if s.dockerComposeStartedByTest {
		err := raw.StopDockerCompose(s.ctx)
		s.NoError(err)
	}
}

func (s *SynchronizerUpdateNodeSuite) TearDownTest() {
	defer os.RemoveAll(s.tempDir)
}

func TestSynchronizerUpdateNodeSuiteSuite(t *testing.T) {
	suite.Run(t, new(SynchronizerUpdateNodeSuite))
}

func (s *SynchronizerUpdateNodeSuite) TestGetFirstRefWithStatusNone() {
	ctx := context.Background()
	s.fillRefData(ctx)
	batchSize := 50
	s.synchronizerUpdate.BatchSize = batchSize
	inputsStatusNone, err := s.synchronizerUpdate.getFirstRefWithStatusNone(ctx)
	s.NoError(err)
	s.NotNil(inputsStatusNone)
	s.Equal("NONE", inputsStatusNone.Status)
}

// Dear Programmer, I hope this message finds you well.
// Keep coding, keep learning, and never forget—your work shapes the future.
func (s *SynchronizerUpdateNodeSuite) TestUpdateInputStatusNotEqNone() {
	ctx := context.Background()
	s.fillRefData(ctx)

	// check setup
	unprocessed := s.countInputWithStatusNone(ctx)
	s.Require().Equal(TOTAL_INPUT_TEST, unprocessed)

	batchSize := s.synchronizerUpdate.BatchSize

	// first call
	err := s.synchronizerUpdate.SyncInputStatus(ctx)
	s.Require().NoError(err)
	first := s.countAcceptedInput(ctx)
	s.Equal(50, batchSize)
	s.Equal(batchSize, first)

	// second call
	err = s.synchronizerUpdate.SyncInputStatus(ctx)
	s.Require().NoError(err)
	none := s.getNoneInputs(ctx)
	slog.Debug("None", "none", none)
	second := s.countAcceptedInput(ctx)
	s.Equal(TOTAL_INPUT_TEST-1, second)
}

func (s *SynchronizerUpdateNodeSuite) countInputWithStatusNone(ctx context.Context) int {
	status := model.STATUS_PROPERTY
	value := fmt.Sprintf("%d", model.CompletionStatusUnprocessed)
	filter := []*model.ConvenienceFilter{
		{
			Field: &status,
			Eq:    &value,
		},
	}
	total, err := s.container.GetInputRepository().Count(ctx, filter)
	s.Require().NoError(err)
	return int(total)
}

func (s *SynchronizerUpdateNodeSuite) getNoneInputs(ctx context.Context) *commons.PageResult[model.AdvanceInput] {
	status := model.STATUS_PROPERTY
	value := fmt.Sprintf("%d", model.CompletionStatusUnprocessed)
	filter := []*model.ConvenienceFilter{
		{
			Field: &status,
			Eq:    &value,
		},
	}
	result, err := s.container.GetInputRepository().FindAll(ctx, nil, nil, nil, nil, filter)
	s.Require().NoError(err)
	return result
}

func (s *SynchronizerUpdateNodeSuite) countAcceptedInput(ctx context.Context) int {
	status := model.STATUS_PROPERTY
	value := fmt.Sprintf("%d", model.CompletionStatusAccepted)
	filter := []*model.ConvenienceFilter{
		{
			Field: &status,
			Eq:    &value,
		},
	}
	total, err := s.container.GetInputRepository().Count(ctx, filter)
	s.Require().NoError(err)
	return int(total)
}

func (s *SynchronizerUpdateNodeSuite) fillRefData(ctx context.Context) {
	appContract := common.HexToAddress(DEFAULT_TEST_APP_CONTRACT)
	msgSender := common.HexToAddress(SenderAddress)
	txCtx, err := s.synchronizerUpdate.startTransaction(ctx)
	s.Require().NoError(err)
	for i := 0; i < TOTAL_INPUT_TEST; i++ {
		id := strconv.FormatInt(int64(i), 10) // our ID
		err := s.container.GetRawInputRepository().Create(txCtx, repository.RawInputRef{
			ID:          id,
			InputIndex:  uint64(i),
			AppID:       uint64(1),
			AppContract: appContract.Hex(),
			Status:      "NONE",
			ChainID:     "31337",
		})
		s.Require().NoError(err)
		_, err = s.container.GetInputRepository().Create(txCtx, model.AdvanceInput{
			ID:          id,
			Index:       i,
			Status:      model.CompletionStatusUnprocessed,
			AppContract: appContract,
			MsgSender:   msgSender,
			ChainId:     "31337",
		})
		s.Require().NoError(err)
	}
	err = s.synchronizerUpdate.commitTransaction(txCtx)
	s.Require().NoError(err)
}
