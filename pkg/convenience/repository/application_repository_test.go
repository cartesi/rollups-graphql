package repository

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	configtest "github.com/cartesi/rollups-graphql/pkg/convenience/config_test"
	"github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

type ApplicationRepositorySuite struct {
	suite.Suite
	repository *ApplicationRepository
	tempDir    string
}

func (s *ApplicationRepositorySuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)

	// Temp
	tempDir, err := os.MkdirTemp("", "")
	s.NoError(err)
	s.tempDir = tempDir

	// Database
	sqliteFileName := filepath.Join(tempDir, "application.sqlite3")

	db := sqlx.MustConnect("sqlite3", sqliteFileName)

	s.repository = &ApplicationRepository{
		Db: *db,
	}
	err = s.repository.CreateTables()
	s.NoError(err)
}

func TestApplicationRepositorySuite(t *testing.T) {
	suite.Run(t, new(ApplicationRepositorySuite))
}

func newApp() *model.ConvenienceApplication {
	return &model.ConvenienceApplication{
		ID:                 1,
		Name:               "app1",
		ApplicationAddress: common.HexToAddress(configtest.DEFAULT_TEST_APP_CONTRACT),
	}
}

func (s *ApplicationRepositorySuite) TestCreateApplication() {
	ctx := context.Background()
	app := newApp()
	_, err := s.repository.Create(ctx, app)
	s.NoError(err)
	count, err := s.repository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(1, int(count))
}

func (s *ApplicationRepositorySuite) TestFindApplication() {
	ctx := context.Background()
	counter := 10
	add := configtest.DEFAULT_TEST_APP_CONTRACT

	for i := 0; i < counter; i++ {
		app := newApp()
		app.Name = fmt.Sprintf("app%d", i)
		_, err := s.repository.Create(ctx, app)
		s.NoError(err)
	}

	key := model.APP_CONTRACT
	filter := []*model.ConvenienceFilter{
		{
			Field: &key,
			Eq:    &add,
		},
	}
	count, err := s.repository.Count(ctx, filter)
	s.NoError(err)
	s.Equal(counter, int(count))

	add = "0xdeadbeef"

	for i := 0; i < counter; i++ {
		app := newApp()
		app.Name = fmt.Sprintf("app%d", i)
		app.ApplicationAddress = common.HexToAddress(add)
		_, err := s.repository.Create(ctx, app)
		s.NoError(err)
	}

	filter = []*model.ConvenienceFilter{
		{
			Field: &key,
			Eq:    &add,
		},
	}
	count, err = s.repository.Count(ctx, filter)
	s.NoError(err)
	s.Equal(counter, int(count))
}
