package repository

import (
	"context"
	"log/slog"
	"testing"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	cModel "github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/suite"
)

//
// Test suite
//

type ReportRepositorySuite struct {
	suite.Suite
	reportRepository *ReportRepository
}

func (s *ReportRepositorySuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	db := sqlx.MustConnect("sqlite3", ":memory:")
	s.reportRepository = &ReportRepository{
		Db: db,
	}
	err := s.reportRepository.CreateTables()
	s.NoError(err)
}

func TestReportRepositorySuite(t *testing.T) {
	suite.Run(t, new(ReportRepositorySuite))
}

func (s *ReportRepositorySuite) TestCreateTables() {
	err := s.reportRepository.CreateTables()
	s.NoError(err)
}

func (s *ReportRepositorySuite) TestCreateReport() {
	ctx := context.Background()
	_, err := s.reportRepository.CreateReport(ctx, cModel.Report{
		Index:      1,
		InputIndex: 2,
		Payload:    "1122",
	})
	s.NoError(err)
}

func (s *ReportRepositorySuite) TestCreateReportAndFind() {
	ctx := context.Background()
	_, err := s.reportRepository.CreateReport(ctx, cModel.Report{
		InputIndex: 1,
		Index:      2,
		Payload:    "1122",
	})
	s.NoError(err)
	report, err := s.reportRepository.FindByInputAndOutputIndex(
		ctx,
		uint64(1),
		uint64(2),
	)
	s.NoError(err)
	s.Equal("0x1122", report.Payload)
}

func (s *ReportRepositorySuite) TestReportNotFound() {
	ctx := context.Background()
	report, err := s.reportRepository.FindByInputAndOutputIndex(
		ctx,
		uint64(404),
		uint64(404),
	)
	s.NoError(err)
	s.Nil(report)
}

func (s *ReportRepositorySuite) TestCreateReportAndFindAll() {
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		for j := 0; j < 4; j++ {
			_, err := s.reportRepository.CreateReport(
				ctx,
				cModel.Report{
					InputIndex: i,
					Index:      j,
					Payload:    "1122",
				})
			s.NoError(err)
		}
	}
	reports, err := s.reportRepository.FindAll(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Equal(12, int(reports.Total))
	s.Equal(0, reports.Rows[0].InputIndex)
	s.Equal(2, reports.Rows[len(reports.Rows)-1].InputIndex)

	filter := []*cModel.ConvenienceFilter{}
	{
		field := "InputIndex"
		value := "1"
		filter = append(filter, &cModel.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	reports, err = s.reportRepository.FindAll(ctx, nil, nil, nil, nil, filter)
	s.NoError(err)
	s.Equal(4, int(reports.Total))
	s.Equal(1, reports.Rows[0].InputIndex)
	s.Equal(0, reports.Rows[0].Index)
	s.Equal(1, reports.Rows[len(reports.Rows)-1].InputIndex)
	s.Equal(3, reports.Rows[len(reports.Rows)-1].Index)
	s.Equal("0x1122", reports.Rows[0].Payload)
}

func (s *ReportRepositorySuite) TestBatchFindAll() {
	ctx := context.Background()
	appContract := common.HexToAddress(ApplicationAddress)
	for i := 0; i < 3; i++ {
		for j := 0; j < 4; j++ {
			_, err := s.reportRepository.CreateReport(
				ctx,
				cModel.Report{
					InputIndex:  i,
					Index:       j,
					Payload:     "0x1122",
					AppContract: appContract,
				})
			s.Require().NoError(err)
		}
	}
	filters := []*BatchFilterItem{
		{
			AppContract: &appContract,
			InputIndex:  0,
		},
	}
	results, err := s.reportRepository.BatchFindAllByInputIndexAndAppContract(
		ctx, filters,
	)
	s.Require().Equal(0, len(err))
	s.Equal(1, len(results))
	s.Equal(4, len(results[0].Rows))
	s.Equal(4, int(results[0].Total))
}

func (r *ReportRepositorySuite) TestFindReportByAppContractAndIndex() {

	ctx := context.Background()
	_, err := r.reportRepository.CreateReport(ctx, cModel.Report{
		Index:       2222,
		InputIndex:  1,
		Payload:     "0x1122",
		AppContract: common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
	})
	r.NoError(err)

	_, err = r.reportRepository.CreateReport(ctx, cModel.Report{
		Index:       3333,
		InputIndex:  2,
		Payload:     "0xFF22",
		AppContract: common.HexToAddress("0xf29Ed6e51bbd88F7F4ce6bA8827389cffFb92255"),
	})
	r.NoError(err)

	report, err := r.reportRepository.FindReportByAppContractAndIndex(ctx, 2, common.HexToAddress("0xf29Ed6e51bbd88F7F4ce6bA8827389cffFb92255"))
	r.NoError(err)

	r.Equal(common.HexToAddress("0xf29Ed6e51bbd88F7F4ce6bA8827389cffFb92255"), report.AppContract)
	r.Equal(3333, report.Index)

}
