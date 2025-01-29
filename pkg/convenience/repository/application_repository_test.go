package repository

import (
	"context"
	"log/slog"
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
}

func (s *ApplicationRepositorySuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	db := sqlx.MustConnect("sqlite3", ":memory:")
	s.repository = &ApplicationRepository{
		Db: *db,
	}
	err := s.repository.CreateTables()
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
	val := *model.ApplicationStateDisabled.String()
	filter := []*model.ConvenienceFilter{
		{
			Field: &key,
			Eq:    &val,
		},
	}
	count, err := s.repository.Count(ctx, filter)
	s.NoError(err)
	s.Equal(1, int(count))
}
