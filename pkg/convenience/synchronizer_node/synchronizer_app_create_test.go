package synchronizernode

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/cartesi/rollups-graphql/v2/pkg/commons"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience/repository"
	"github.com/cartesi/rollups-graphql/v2/postgres/raw"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

type SynchronizerAppCreate struct {
	suite.Suite
	ctx                        context.Context
	dockerComposeStartedByTest bool
	tempDir                    string
	container                  *convenience.Container
	synchronizerAppCreator     *SynchronizerAppCreator
	appRepository              *repository.ApplicationRepository
	rawNodeV2Repository        *RawRepository
}

func (s *SynchronizerAppCreate) SetupSuite() {
	pgUp := commons.IsPortInUse(5432)
	if !pgUp {
		err := raw.RunDockerCompose(s.ctx)
		s.NoError(err)
		s.dockerComposeStartedByTest = true
	}
}

func (s *SynchronizerAppCreate) SetupTest() {
	s.ctx = context.Background()
	commons.ConfigureLog(slog.LevelDebug)

	// Temp
	tempDir, err := os.MkdirTemp("", "")
	s.NoError(err)
	s.tempDir = tempDir

	// Database
	sqliteFileName := filepath.Join(tempDir, "application.sqlite3")

	db := sqlx.MustConnect("sqlite3", sqliteFileName)
	s.container = convenience.NewContainer(db, false)

	dbNodeV2 := sqlx.MustConnect("postgres", RAW_DB_URL)
	s.rawNodeV2Repository = NewRawRepository(RAW_DB_URL, dbNodeV2)

	s.appRepository = s.container.GetApplicationRepository(s.ctx)

	s.synchronizerAppCreator = NewSynchronizerAppCreator(
		s.appRepository,
		s.rawNodeV2Repository,
	)
}

func (s *SynchronizerAppCreate) TearDownTest() {
	defer os.RemoveAll(s.tempDir)
}

func TestSynchronizerAppCreateSuite(t *testing.T) {
	suite.Run(t, new(SynchronizerAppCreate))
}

func (s *SynchronizerAppCreate) TestAppCreate() {
	apps, err := s.rawNodeV2Repository.FindAllAppsRef(s.ctx)
	s.Require().NoError(err)
	s.Require().NotEmpty(apps)
	// size := min(LIMIT, uint64(len(apps)))
	size := 1

	firstApp := apps[0]
	s.Equal("echo-dapp", firstApp.Name)
	s.Equal(DEFAULT_TEST_APP_CONTRACT, firstApp.ApplicationAddress.Hex())

	for i := 0; i < 2; i++ {
		err = s.synchronizerAppCreator.SyncApps(s.ctx)
		s.Require().NoError(err)
		count, err := s.appRepository.Count(s.ctx, nil)
		s.Require().NoError(err)
		s.Require().Equal(size, int(count))
	}
}
