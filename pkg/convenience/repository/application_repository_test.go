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

func (s *ApplicationRepositorySuite) TestCreateApplication() {
	ctx := context.Background()
	_, err := s.repository.Create(ctx, &model.ConvenienceApplication{
		ID:                   1,
		Name:                 "app1",
		TemplateHash:         []byte("0x1234"),
		TemplateURI:          "http://template.com",
		EpochLength:          100,
		State:                "active",
		LastProcessedBlock:   0,
		LastClaimCheckBlock:  0,
		LastOutputCheckBlock: 0,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
		ProcessedInputs:      0,
		ApplicationAddress:   common.HexToAddress(configtest.DEFAULT_TEST_APP_CONTRACT),
		ConsensusAddress:     common.HexToAddress(configtest.DEFAULT_TEST_APP_CONTRACT),
	})
	s.NoError(err)
	count, err := s.repository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(1, int(count))
}
