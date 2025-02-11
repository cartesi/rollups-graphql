package convenience

import (
	"fmt"

	"github.com/cartesi/rollups-graphql/pkg/convenience/decoder"
	"github.com/cartesi/rollups-graphql/pkg/convenience/repository"
	"github.com/cartesi/rollups-graphql/pkg/convenience/services"
	"github.com/cartesi/rollups-graphql/pkg/convenience/synchronizer"
	"github.com/jmoiron/sqlx"
)

// what is the best DI/IoC framework for go?

type Container struct {
	db                     *sqlx.DB
	outputDecoder          *decoder.OutputDecoder
	convenienceService     *services.ConvenienceService
	outputRepository       *repository.OutputRepository
	repository             *repository.VoucherRepository
	syncRepository         *repository.SynchronizerRepository
	graphQLSynchronizer    *synchronizer.Synchronizer
	voucherFetcher         *synchronizer.VoucherFetcher
	noticeRepository       *repository.NoticeRepository
	inputRepository        *repository.InputRepository
	reportRepository       *repository.ReportRepository
	AutoCount              bool
	rawInputRefRepository  *repository.RawInputRefRepository
	rawOutputRefRepository *repository.RawOutputRefRepository
	appRepository          *repository.ApplicationRepository
}

func NewContainer(db sqlx.DB, autoCount bool) *Container {
	return &Container{
		db:        &db,
		AutoCount: autoCount,
	}
}

func (c *Container) GetApplicationRepository() *repository.ApplicationRepository {
	if c.appRepository != nil {
		return c.appRepository
	}
	c.appRepository = &repository.ApplicationRepository{
		Db: *c.db,
	}
	err := c.appRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	return c.appRepository
}

func (c *Container) GetOutputDecoder() *decoder.OutputDecoder {
	if c.outputDecoder != nil {
		return c.outputDecoder
	}
	c.outputDecoder = decoder.NewOutputDecoder(*c.GetConvenienceService())
	return c.outputDecoder
}

func (c *Container) GetOutputRepository() *repository.OutputRepository {
	if c.outputRepository != nil {
		return c.outputRepository
	}
	c.outputRepository = &repository.OutputRepository{
		Db: *c.db,
	}
	return c.outputRepository
}

func (c *Container) GetVoucherRepository() *repository.VoucherRepository {
	if c.repository != nil {
		return c.repository
	}
	c.repository = &repository.VoucherRepository{
		Db:               *c.db,
		OutputRepository: *c.GetOutputRepository(),
		AutoCount:        c.AutoCount,
	}
	err := c.repository.CreateTables()
	if err != nil {
		panic(err)
	}
	return c.repository
}

func (c *Container) GetSyncRepository() *repository.SynchronizerRepository {
	if c.syncRepository != nil {
		return c.syncRepository
	}
	c.syncRepository = &repository.SynchronizerRepository{
		Db: *c.db,
	}
	err := c.syncRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	return c.syncRepository
}

func (c *Container) GetNoticeRepository() *repository.NoticeRepository {
	if c.noticeRepository != nil {
		return c.noticeRepository
	}
	c.noticeRepository = &repository.NoticeRepository{
		Db:               *c.db,
		OutputRepository: *c.GetOutputRepository(),
		AutoCount:        c.AutoCount,
	}
	err := c.noticeRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	return c.noticeRepository
}

func (c *Container) GetRawInputRepository() *repository.RawInputRefRepository {
	if c.rawInputRefRepository != nil {
		return c.rawInputRefRepository
	}
	c.rawInputRefRepository = &repository.RawInputRefRepository{
		Db: *c.db,
	}
	err := c.rawInputRefRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	return c.rawInputRefRepository
}

func (c *Container) GetRawOutputRefRepository() *repository.RawOutputRefRepository {
	if c.db == nil {
		panic(fmt.Errorf("db cannot be nil"))
	}
	if c.rawOutputRefRepository != nil {
		return c.rawOutputRefRepository
	}
	c.rawOutputRefRepository = &repository.RawOutputRefRepository{
		Db: c.db,
	}
	err := c.rawOutputRefRepository.CreateTable()
	if err != nil {
		panic(err)
	}
	return c.rawOutputRefRepository
}

func (c *Container) GetInputRepository() *repository.InputRepository {
	if c.inputRepository != nil {
		return c.inputRepository
	}
	c.inputRepository = &repository.InputRepository{
		Db: *c.db,
	}
	err := c.inputRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	return c.inputRepository
}

func (c *Container) GetReportRepository() *repository.ReportRepository {
	if c.reportRepository != nil {
		return c.reportRepository
	}
	c.reportRepository = &repository.ReportRepository{
		Db:        c.db,
		AutoCount: c.AutoCount,
	}
	err := c.reportRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	return c.reportRepository
}

func (c *Container) GetConvenienceService() *services.ConvenienceService {
	if c.convenienceService != nil {
		return c.convenienceService
	}
	c.convenienceService = services.NewConvenienceService(
		c.GetVoucherRepository(),
		c.GetNoticeRepository(),
		c.GetInputRepository(),
		c.GetReportRepository(),
		c.GetApplicationRepository(),
	)
	return c.convenienceService
}

func (c *Container) GetGraphQLSynchronizer() *synchronizer.Synchronizer {
	if c.graphQLSynchronizer != nil {
		return c.graphQLSynchronizer
	}
	c.graphQLSynchronizer = synchronizer.NewSynchronizer(
		c.GetOutputDecoder(),
		c.GetVoucherFetcher(),
		c.GetSyncRepository(),
	)
	return c.graphQLSynchronizer
}

func (c *Container) GetVoucherFetcher() *synchronizer.VoucherFetcher {
	if c.voucherFetcher != nil {
		return c.voucherFetcher
	}
	c.voucherFetcher = synchronizer.NewVoucherFetcher()
	return c.voucherFetcher
}
