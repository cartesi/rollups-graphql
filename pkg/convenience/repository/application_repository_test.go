package repository

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

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
		ID:                   1,
		Name:                 "app1",
		TemplateHash:         []byte("0x1234"),
		TemplateURI:          "http://template.com",
		EpochLength:          100,
		State:                *model.ApplicationStateEnabled.String(),
		LastProcessedBlock:   0,
		LastClaimCheckBlock:  0,
		LastOutputCheckBlock: 0,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
		ProcessedInputs:      0,
		ApplicationAddress:   common.HexToAddress(configtest.DEFAULT_TEST_APP_CONTRACT),
		ConsensusAddress:     common.HexToAddress(configtest.DEFAULT_TEST_APP_CONTRACT),
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

func (s *ApplicationRepositorySuite) TestUpdateApplication() {
	ctx := context.Background()
	app := newApp()
	_, err := s.repository.Create(ctx, app)
	s.NoError(err)
	app.State = *model.ApplicationStateDisabled.String()
	err = s.repository.Update(ctx, app)
	s.NoError(err)
	key := model.STATE
	filter := []*model.ConvenienceFilter{
		{
			Field: &key,
			Eq:    model.ApplicationStateDisabled.String(),
		},
	}
	count, err := s.repository.Count(ctx, filter)
	s.NoError(err)
	s.Equal(1, int(count))
}

func (s *ApplicationRepositorySuite) TestFindApplication() {
	ctx := context.Background()

	be_enabled := 5
	be_disabled := 3
	counter := 0

	for ; counter < be_enabled; counter++ {
		app := newApp()
		app.Name = fmt.Sprintf("app%d", counter)
		_, err := s.repository.Create(ctx, app)
		s.NoError(err)
	}

	key := model.STATE
	filter := []*model.ConvenienceFilter{
		{
			Field: &key,
			Eq:    model.ApplicationStateEnabled.String(),
		},
	}
	count, err := s.repository.Count(ctx, filter)
	s.NoError(err)
	s.Equal(be_enabled, int(count))

	for i := 0; i < be_disabled; i++ {
		app := newApp()
		app.Name = fmt.Sprintf("app%d", counter)
		app.State = *model.ApplicationStateDisabled.String()
		_, err := s.repository.Create(ctx, app)
		s.NoError(err)
		counter++
	}

	filter = []*model.ConvenienceFilter{
		{
			Field: &key,
			Eq:    model.ApplicationStateDisabled.String(),
		},
	}
	disabled, err := s.repository.FindAll(ctx, filter)
	s.NoError(err)
	s.Equal(be_disabled, len(disabled))
	for _, app := range disabled {
		s.Equal(*model.ApplicationStateDisabled.String(), app.State)
	}
}
