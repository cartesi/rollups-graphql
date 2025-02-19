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
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

type ApplicationRepositorySuite struct {
	suite.Suite
	repository *ApplicationRepository
	tempDir    string
}

func (a *ApplicationRepositorySuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)

	// Temp
	tempDir, err := os.MkdirTemp("", "")
	a.NoError(err)
	a.tempDir = tempDir

	// Database
	sqliteFileName := filepath.Join(tempDir, "application.sqlite3")

	db := sqlx.MustConnect("sqlite3", sqliteFileName)

	a.repository = &ApplicationRepository{
		Db: *db,
	}
	err = a.repository.CreateTables()
	a.NoError(err)
}

func (a *ApplicationRepositorySuite) TearDownTest() {
	err := a.repository.Db.Close()
	a.NoError(err)
	err = os.RemoveAll(a.tempDir)
	a.NoError(err)
}

func TestApplicationRepositorySuite(t *testing.T) {
	suite.Run(t, new(ApplicationRepositorySuite))
}

func newApp() *model.ConvenienceApplication {
	address := configtest.DEFAULT_TEST_APP_CONTRACT
	return &model.ConvenienceApplication{
		ID:                 1,
		Name:               "app1",
		ApplicationAddress: address,
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
	anotherAppContract := configtest.DEFAULT_TEST_APP_CONTRACT

	for i := 0; i < counter; i++ {
		app := newApp()
		app.Name = fmt.Sprintf("app%d", i)
		app.ApplicationAddress = anotherAppContract
		_, err := s.repository.Create(ctx, app)
		s.NoError(err)
	}

	key := model.APP_CONTRACT
	filter := []*model.ConvenienceFilter{
		{
			Field: &key,
			Eq:    &anotherAppContract,
		},
	}
	count, err := s.repository.Count(ctx, filter)
	s.NoError(err)
	s.Equal(counter, int(count))

	anotherAppContract = "0x544a3B76B84b1E98c13437A1591E713Dd314387F"

	for i := 0; i < counter; i++ {
		app := newApp()
		app.Name = fmt.Sprintf("app%d", i)
		app.ApplicationAddress = anotherAppContract
		_, err := s.repository.Create(ctx, app)
		s.NoError(err)
	}

	filter = []*model.ConvenienceFilter{
		{
			Field: &key,
			Eq:    &anotherAppContract,
		},
	}
	count, err = s.repository.Count(ctx, filter)
	s.NoError(err)
	s.Equal(counter, int(count))
}
