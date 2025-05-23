package synchronizernode

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/cartesi/rollups-graphql/v2/pkg/commons"
	"github.com/cartesi/rollups-graphql/v2/pkg/contracts"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience"
	"github.com/cartesi/rollups-graphql/v2/postgres/raw"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

type SynchronizerInputCreate struct {
	suite.Suite
	ctx                        context.Context
	dockerComposeStartedByTest bool
	tempDir                    string
	container                  *convenience.Container
	synchronizerInputCreate    *SynchronizerInputCreator
	rawNodeV2Repository        *RawRepository
}

func (s *SynchronizerInputCreate) SetupSuite() {
	pgUp := commons.IsPortInUse(5432)
	if !pgUp {
		err := raw.RunDockerCompose(s.ctx)
		s.NoError(err)
		s.dockerComposeStartedByTest = true
	}
}

func (s *SynchronizerInputCreate) SetupTest() {
	s.ctx = context.Background()
	commons.ConfigureLog(slog.LevelDebug)

	// Temp
	tempDir, err := os.MkdirTemp("", "")
	s.NoError(err)
	s.tempDir = tempDir

	// Database
	sqliteFileName := filepath.Join(tempDir, "output.sqlite3")

	db := sqlx.MustConnect("sqlite3", sqliteFileName)
	s.container = convenience.NewContainer(db, false)

	dbNodeV2 := sqlx.MustConnect("postgres", RAW_DB_URL)
	s.rawNodeV2Repository = NewRawRepository(RAW_DB_URL, dbNodeV2)

	abi, err := contracts.InputsMetaData.GetAbi()
	if err != nil {
		s.Require().NoError(err)
	}
	abiDecoder := NewAbiDecoder(abi)
	s.synchronizerInputCreate = NewSynchronizerInputCreator(
		s.container.GetInputRepository(s.ctx),
		s.container.GetRawInputRepository(s.ctx),
		s.rawNodeV2Repository,
		abiDecoder,
	)
}

func (s *SynchronizerInputCreate) TearDownTest() {
	defer os.RemoveAll(s.tempDir)
}

func TestSynchronizerInputCreateSuite(t *testing.T) {
	suite.Run(t, new(SynchronizerInputCreate))
}

func (s *SynchronizerInputCreate) TestGetAdvanceInputFromMap() {
	inputs, err := s.rawNodeV2Repository.FindAllRawInputs(s.ctx)
	s.Require().NoError(err)

	rawInput := inputs[0]
	advanceInput, err := s.synchronizerInputCreate.GetAdvanceInputFromMap(rawInput)
	s.Require().NoError(err)
	s.Equal(DEFAULT_TEST_APP_CONTRACT, advanceInput.AppContract.Hex())
	s.Equal(0, advanceInput.Index)
	s.Equal(0, advanceInput.InputBoxIndex)
	s.Equal("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", advanceInput.MsgSender.Hex())
	s.Greater(int(advanceInput.BlockNumber), 42, "Block number") // nolint
	s.Equal("13370", advanceInput.ChainId)
	s.Equal(commons.ConvertStatusStringToCompletionStatus("ACCEPTED"), advanceInput.Status)
	minBlockTimestamp := int64(1743106900) // nolint
	s.Greater(advanceInput.BlockTimestamp.Unix(), minBlockTimestamp)
}

func (s *SynchronizerInputCreate) TestCreateInputs() {
	ctx := context.Background()

	// check setup
	proofCount := s.countOurInputs(ctx)
	s.Require().Equal(0, proofCount)

	// first call
	err := s.synchronizerInputCreate.SyncInputs(ctx)
	s.Require().NoError(err)
	first := s.countOurInputs(ctx)
	s.Equal(TOTAL_INPUT_TEST/2, first)

	// second call
	err = s.synchronizerInputCreate.SyncInputs(ctx)
	s.Require().NoError(err)

	err = s.synchronizerInputCreate.SyncInputs(ctx)
	s.Require().NoError(err)
	second := s.countOurInputs(ctx)
	s.Equal(TOTAL_INPUT_TEST+1, second)
}

func (s *SynchronizerInputCreate) countOurInputs(ctx context.Context) int {
	total, err := s.container.GetInputRepository(s.ctx).Count(ctx, nil)
	s.Require().NoError(err)
	return int(total)
}
