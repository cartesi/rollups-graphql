package repository

import (
	"context"
	"log/slog"
	"testing"

	"github.com/cartesi/rollups-graphql/v2/pkg/commons"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience/model"
	"github.com/jmoiron/sqlx"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/suite"
)

type SynchronizerRepositorySuite struct {
	suite.Suite
	repository *SynchronizerRepository
	db         *sqlx.DB
	ctx        context.Context
	ctxCancel  context.CancelFunc
}

func (s *SynchronizerRepositorySuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	s.ctx, s.ctxCancel = context.WithCancel(context.Background())
	s.db = sqlx.MustConnect("sqlite3", ":memory:")
	s.repository = &SynchronizerRepository{
		Db: *s.db,
	}
	err := s.repository.CreateTables(s.ctx)
	s.NoError(err)
}

func (s *SynchronizerRepositorySuite) TearDownTest() {
	s.ctxCancel()
	s.db.Close()
}

func TestSynchronizerRepositorySuiteSuite(t *testing.T) {
	suite.Run(t, new(SynchronizerRepositorySuite))
}

func (s *SynchronizerRepositorySuite) TestCreateSyncFetch() {
	ctx := context.Background()
	_, err := s.repository.Create(ctx, &model.SynchronizerFetch{})
	s.NoError(err)
	count, err := s.repository.Count(ctx)
	s.NoError(err)
	s.Equal(1, int(count))
}

func (s *SynchronizerRepositorySuite) TestGetLastFetched() {
	ctx := context.Background()
	_, err := s.repository.Create(ctx, &model.SynchronizerFetch{})
	s.NoError(err)
	_, err = s.repository.Create(ctx, &model.SynchronizerFetch{})
	s.NoError(err)
	lastFetch, err := s.repository.GetLastFetched(ctx)
	s.NoError(err)
	s.Equal(2, int(lastFetch.Id))
}

func (s *SynchronizerRepositorySuite) TestPurgeOldData() {
	ctx := context.Background()
	var (
		timeAfter  uint64 = 99
		timeBefore uint64 = timeAfter + 1
	)
	_, err := s.repository.Create(ctx, &model.SynchronizerFetch{
		TimestampAfter: timeAfter,
	})
	s.NoError(err)

	err = s.repository.PurgeData(ctx, timeBefore)
	s.NoError(err)

	count, err := s.repository.Count(ctx)
	s.NoError(err)
	s.Equal(1, int(count))
}
