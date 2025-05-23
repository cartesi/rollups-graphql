package repository

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/cartesi/rollups-graphql/v2/pkg/commons"
	configtest "github.com/cartesi/rollups-graphql/v2/pkg/convenience/config_test"
	"github.com/cartesi/rollups-graphql/v2/postgres/raw"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
)

type RawInputRefSuite struct {
	suite.Suite
	inputRepository       *InputRepository
	RawInputRefRepository *RawInputRefRepository
	dbFactory             *commons.DbFactory
	ctx                   context.Context
	db                    *sqlx.DB
	ctxCancel             context.CancelFunc
}

func TestRawRefInputSuite(t *testing.T) {
	suite.Run(t, new(RawInputRefSuite))
}

func (s *RawInputRefSuite) TearDownTest() {
	s.ctxCancel()
	err := s.db.Close()
	s.NoError(err)
	s.dbFactory.Cleanup(s.ctx)
}

func (s *RawInputRefSuite) SetupTest() {
	var err error
	s.ctx, s.ctxCancel = context.WithCancel(context.Background())
	commons.ConfigureLog(slog.LevelDebug)
	s.dbFactory, err = commons.NewDbFactory()
	s.Require().NoError(err)
	s.db = s.dbFactory.CreateDb(s.ctx, "input.sqlite3")
	s.inputRepository = &InputRepository{
		Db: s.db,
	}
	s.RawInputRefRepository = &RawInputRefRepository{
		Db: s.db,
	}

	err = s.inputRepository.CreateTables(s.ctx)
	s.NoError(err)
	err = s.RawInputRefRepository.CreateTables(s.ctx)
	s.NoError(err)
}

func (s *RawInputRefSuite) TestNoDuplicateInputs() {
	ctx := context.Background()
	appContract := common.HexToAddress(configtest.DEFAULT_TEST_APP_CONTRACT)
	err := s.RawInputRefRepository.Create(ctx, RawInputRef{
		ID:          "001",
		AppID:       1,
		InputIndex:  uint64(3),
		AppContract: appContract.Hex(),
		Status:      "NONE",
		ChainID:     "31337",
	})

	s.Require().NoError(err)

	err = s.RawInputRefRepository.Create(ctx, RawInputRef{
		ID:          "001",
		AppID:       1,
		InputIndex:  uint64(3),
		AppContract: appContract.Hex(),
		Status:      "NONE",
		ChainID:     "31337",
	})
	s.Require().NoError(err)

	var count int
	err = s.RawInputRefRepository.Db.QueryRow(`SELECT COUNT(*) FROM convenience_input_raw_references WHERE input_index = ? AND app_contract = ?`,
		uint64(3), appContract.Hex()).Scan(&count)

	s.Require().NoError(err)
	s.Require().Equal(1, count)
}

func (s *RawInputRefSuite) TestSaveDifferentInputs() {
	ctx := context.Background()
	appContract := common.HexToAddress(configtest.DEFAULT_TEST_APP_CONTRACT)
	err := s.RawInputRefRepository.Create(ctx, RawInputRef{
		ID:          "001",
		AppID:       uint64(1),
		InputIndex:  uint64(0),
		AppContract: appContract.Hex(),
		Status:      "NONE",
		ChainID:     "31337",
	})

	s.Require().NoError(err)

	err = s.RawInputRefRepository.Create(ctx, RawInputRef{
		ID:          "002",
		AppID:       uint64(1),
		InputIndex:  uint64(1),
		AppContract: appContract.Hex(),
		Status:      "NONE",
		ChainID:     "31337",
	})
	s.Require().NoError(err)

	var count int
	err = s.RawInputRefRepository.Db.QueryRow(`SELECT COUNT(*) FROM convenience_input_raw_references`).Scan(&count)

	s.Require().NoError(err)
	s.Require().Equal(2, count)
}

func (s *RawInputRefSuite) TestFindByInputIndexAndAppContract() {
	ctx := context.Background()
	appContract := common.HexToAddress(configtest.DEFAULT_TEST_APP_CONTRACT)
	err := s.RawInputRefRepository.Create(ctx, RawInputRef{
		ID:          "001",
		InputIndex:  uint64(1),
		AppContract: appContract.Hex(),
		Status:      "NONE",
		ChainID:     "31337",
	})

	s.Require().NoError(err)

	input, err := s.RawInputRefRepository.FindByInputIndexAndAppContract(ctx, uint64(1), &appContract)

	s.Require().NoError(err)
	s.Require().Equal("001", input.ID)
	s.Require().Equal("NONE", input.Status)
	s.Require().Equal("31337", input.ChainID)
	s.Require().Equal(appContract.Hex(), input.AppContract)
}

func (s *RawInputRefSuite) TestUpdateStatusJustOneRawID() {
	ctx := context.Background()
	appContract := common.HexToAddress(configtest.DEFAULT_TEST_APP_CONTRACT)
	err := s.RawInputRefRepository.Create(ctx, RawInputRef{
		ID:          "001",
		InputIndex:  uint64(1),
		AppID:       uint64(3),
		AppContract: appContract.Hex(),
		Status:      "NONE",
		ChainID:     "31337",
	})

	s.Require().NoError(err)
	rawInputsRefs := []RawInputRef{
		{
			ID:          "001",
			InputIndex:  uint64(1),
			AppID:       uint64(3),
			AppContract: "someContractAddress",
			Status:      "NONE",
			ChainID:     "31337",
		},
	}
	err = s.RawInputRefRepository.UpdateStatus(ctx, rawInputsRefs, "ACCEPTED")
	s.Require().NoError(err)
}

func (s *RawInputRefSuite) TestUpdateStatusJustOneRawIDUsingPG() {
	s.setupPG(s.ctx)
	ctx := context.Background()
	appContract := common.HexToAddress(configtest.DEFAULT_TEST_APP_CONTRACT)
	err := s.RawInputRefRepository.Create(ctx, RawInputRef{
		ID:          "001",
		InputIndex:  uint64(1),
		AppID:       uint64(7),
		AppContract: appContract.Hex(),
		Status:      "NONE",
		ChainID:     "31337",
	})

	s.Require().NoError(err)
	rawInputsRefs := []RawInputRef{
		{
			ID:          "001",
			InputIndex:  uint64(1),
			AppID:       uint64(7),
			AppContract: appContract.Hex(),
			Status:      "NONE",
			ChainID:     "31337",
		},
	}
	err = s.RawInputRefRepository.UpdateStatus(ctx, rawInputsRefs, "ACCEPTED")
	s.Require().NoError(err)
	rawInputRef, err := s.RawInputRefRepository.FindByInputIndexAndAppContract(ctx, uint64(1), &appContract)
	s.Require().NoError(err)
	s.Equal("ACCEPTED", rawInputRef.Status)
}

func (s *RawInputRefSuite) setupPG(ctx context.Context) {
	envMap, err := raw.LoadMapEnvFile()
	s.NoError(err)
	dbName := "rollupsdb"
	dbPass := "password"
	if _, ok := envMap["POSTGRES_PASSWORD"]; ok {
		dbPass = envMap["POSTGRES_PASSWORD"]
	}
	if _, ok := envMap["POSTGRES_DB"]; ok {
		dbName = envMap["POSTGRES_DB"]
	}
	uri := fmt.Sprintf("postgres://postgres:%s@localhost:5432/%s?sslmode=disable", dbPass, dbName)
	slog.Info("Raw Input URI", "uri", uri)
	dbNodeV2 := sqlx.MustConnect("postgres", uri)

	s.RawInputRefRepository = &RawInputRefRepository{
		Db: dbNodeV2,
	}

	err = s.RawInputRefRepository.CreateTables(ctx)
	s.NoError(err)
}
