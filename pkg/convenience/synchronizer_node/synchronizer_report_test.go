package synchronizernode

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/cartesi/rollups-graphql/v2/pkg/commons"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience"
	"github.com/cartesi/rollups-graphql/v2/postgres/raw"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

type SynchronizerReportSuite struct {
	suite.Suite
	ctx                        context.Context
	ctxCancel                  context.CancelFunc
	db                         *sqlx.DB
	dbNodeV2                   *sqlx.DB
	dockerComposeStartedByTest bool
	tempDir                    string
	container                  *convenience.Container
	synchronizerReport         *SynchronizerReport
	rawNode                    *RawRepository
}

func (s *SynchronizerReportSuite) SetupSuite() {
	pgUp := commons.IsPortInUse(5432)
	if !pgUp {
		err := raw.RunDockerCompose(s.ctx)
		s.NoError(err)
		s.dockerComposeStartedByTest = true
	}
}

func (s *SynchronizerReportSuite) SetupTest() {
	s.ctx, s.ctxCancel = context.WithCancel(context.Background())
	commons.ConfigureLog(slog.LevelDebug)

	// Temp
	tempDir, err := os.MkdirTemp("", "")
	s.NoError(err)
	s.tempDir = tempDir

	// Database
	sqliteFileName := filepath.Join(tempDir, "report.sqlite3")

	s.db = sqlx.MustConnect("sqlite3", sqliteFileName)
	s.container = convenience.NewContainer(s.db, false)

	s.dbNodeV2 = sqlx.MustConnect("postgres", RAW_DB_URL)
	s.rawNode = NewRawRepository(RAW_DB_URL, s.dbNodeV2)
	s.synchronizerReport = NewSynchronizerReport(
		s.container.GetReportRepository(s.ctx),
		s.rawNode,
	)
}

func (s *SynchronizerReportSuite) TearDownSuite() {
	if s.dockerComposeStartedByTest {
		err := raw.StopDockerCompose(s.ctx)
		s.NoError(err)
	}
}

func (s *SynchronizerReportSuite) TearDownTest() {
	s.ctxCancel()
	err := s.db.Close()
	s.NoError(err)
	err = s.dbNodeV2.Close()
	s.NoError(err)
	os.RemoveAll(s.tempDir)
}

func TestSynchronizerReportSuiteSuite(t *testing.T) {
	suite.Run(t, new(SynchronizerReportSuite))
}

// Dear Programmer, I hope this message finds you well.
// Keep coding, keep learning, and never forget—your work shapes the future.
func (s *SynchronizerReportSuite) TestCreateAllReports() {
	// check setup
	startReportCount := s.countHLReports(s.ctx)
	s.Require().Equal(0, startReportCount)

	// first call
	err := s.synchronizerReport.SyncReports(s.ctx)
	s.Require().NoError(err)

	// second call
	err = s.synchronizerReport.SyncReports(s.ctx)
	s.Require().NoError(err)
	second := s.countHLReports(s.ctx)
	s.Equal(TOTAL_INPUT_TEST, second)
}

func (s *SynchronizerReportSuite) countHLReports(ctx context.Context) int {
	total, err := s.container.GetReportRepository(ctx).Count(ctx, nil)
	s.Require().NoError(err)
	return int(total)
}
