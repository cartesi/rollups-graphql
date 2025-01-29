package synchronizernode

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/cartesi/rollups-graphql/pkg/contracts"
	"github.com/cartesi/rollups-graphql/postgres/raw"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
)

type RawNodeSuite struct {
	suite.Suite
	rawRepository              RawRepository
	ctx                        context.Context
	dockerComposeStartedByTest bool
	DefaultTimeout             time.Duration
}

func (s *RawNodeSuite) SetupSuite() {
	s.DefaultTimeout = 1 * time.Minute
	s.ctx = context.Background()
	pgUp := commons.IsPortInUse(5432)
	if !pgUp {
		err := raw.RunDockerCompose(s.ctx)
		s.NoError(err)
		s.dockerComposeStartedByTest = true
	}

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
	s.rawRepository = RawRepository{
		connectionURL: uri,
		Db:            dbNodeV2,
	}
}

func (s *RawNodeSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
}
func (s *RawNodeSuite) TearDownSuite() {
	if s.dockerComposeStartedByTest {
		err := raw.StopDockerCompose(s.ctx)
		s.NoError(err)
	}
}

func (s *RawNodeSuite) TearDownTest() {}

func TestRawNodeSuite(t *testing.T) {
	suite.Run(t, new(RawNodeSuite))
}

func (s *RawNodeSuite) TestDecodeChainIDFromInputbox() {
	abi, err := contracts.InputsMetaData.GetAbi()
	s.NoError(err)

	rawData := "0x415bf3630000000000000000000000000000000000000000000000000000000000007a690000000000000000000000005112cf49f2511ac7b13a032c4c62a48410fc28fb000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266000000000000000000000000000000000000000000000000000000000000046900000000000000000000000000000000000000000000000000000000670931c70a06511d13afecb37c88e47c1a7357e42205ac4b8e49fcd4632373e036261e26000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000005deadbeef11000000000000000000000000000000000000000000000000000000" // nolint

	// EvmAdvance
	data := common.Hex2Bytes(strings.TrimPrefix(rawData, "0x"))
	methodId := data[:4]
	slog.Debug("MethodId", "methodId", methodId, "hex", rawData[2:10])
	input, err := abi.MethodById(methodId)
	s.NoError(err)

	dataDecoded := make(map[string]interface{})
	dataEncoded := data[4:]
	err = input.Inputs.UnpackIntoMap(dataDecoded, dataEncoded)
	s.NoError(err)
	s.NotEmpty(dataDecoded)
	s.Equal(big.NewInt(31337), dataDecoded["chainId"])
	slog.Info("DataDecoded", "dataDecoded", dataDecoded)
	// s.NotNil(nil)
}

// deprecated
func (s *RawNodeSuite) TestSynchronizerNodeFindInputByOutput() {
	ctx := context.Background()

	input, err := s.rawRepository.FindInputByOutput(ctx, FilterID{IDgt: 1})
	s.NoError(err)
	s.Equal(uint64(1), input.Index)
}
