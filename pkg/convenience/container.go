package convenience

import (
	"context"
	"fmt"

	"github.com/cartesi/rollups-graphql/v2/pkg/convenience/decoder"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience/repository"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience/services"
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
	noticeRepository       *repository.NoticeRepository
	inputRepository        *repository.InputRepository
	reportRepository       *repository.ReportRepository
	AutoCount              bool
	rawInputRefRepository  *repository.RawInputRefRepository
	rawOutputRefRepository *repository.RawOutputRefRepository
	appRepository          *repository.ApplicationRepository
}

func NewContainer(db *sqlx.DB, autoCount bool) *Container {
	return &Container{
		db:        db,
		AutoCount: autoCount,
	}
}

func (c *Container) GetApplicationRepository(ctx context.Context) *repository.ApplicationRepository {
	if c.appRepository != nil {
		return c.appRepository
	}
	c.appRepository = &repository.ApplicationRepository{
		Db: c.db,
	}
	err := c.appRepository.CreateTables(ctx)
	if err != nil {
		panic(err)
	}
	return c.appRepository
}

func (c *Container) GetOutputDecoder(ctx context.Context) *decoder.OutputDecoder {
	if c.outputDecoder != nil {
		return c.outputDecoder
	}
	c.outputDecoder = decoder.NewOutputDecoder(*c.GetConvenienceService(ctx))
	return c.outputDecoder
}

func (c *Container) GetOutputRepository() *repository.OutputRepository {
	if c.outputRepository != nil {
		return c.outputRepository
	}
	c.outputRepository = &repository.OutputRepository{
		Db: c.db,
	}
	return c.outputRepository
}

func (c *Container) GetVoucherRepository(ctx context.Context) *repository.VoucherRepository {
	if c.repository != nil {
		return c.repository
	}
	c.repository = &repository.VoucherRepository{
		Db:               c.db,
		OutputRepository: *c.GetOutputRepository(),
		AutoCount:        c.AutoCount,
	}
	err := c.repository.CreateTables(ctx)
	if err != nil {
		panic(err)
	}
	return c.repository
}

func (c *Container) GetSyncRepository(ctx context.Context) *repository.SynchronizerRepository {
	if c.syncRepository != nil {
		return c.syncRepository
	}
	c.syncRepository = &repository.SynchronizerRepository{
		Db: *c.db,
	}
	err := c.syncRepository.CreateTables(ctx)
	if err != nil {
		panic(err)
	}
	return c.syncRepository
}

func (c *Container) GetNoticeRepository(ctx context.Context) *repository.NoticeRepository {
	if c.noticeRepository != nil {
		return c.noticeRepository
	}
	c.noticeRepository = &repository.NoticeRepository{
		Db:               c.db,
		OutputRepository: *c.GetOutputRepository(),
		AutoCount:        c.AutoCount,
	}
	err := c.noticeRepository.CreateTables(ctx)
	if err != nil {
		panic(err)
	}
	return c.noticeRepository
}

func (c *Container) GetRawInputRepository(ctx context.Context) *repository.RawInputRefRepository {
	if c.rawInputRefRepository != nil {
		return c.rawInputRefRepository
	}
	c.rawInputRefRepository = &repository.RawInputRefRepository{
		Db: c.db,
	}
	err := c.rawInputRefRepository.CreateTables(ctx)
	if err != nil {
		panic(err)
	}
	return c.rawInputRefRepository
}

func (c *Container) GetRawOutputRefRepository(ctx context.Context) *repository.RawOutputRefRepository {
	if c.db == nil {
		panic(fmt.Errorf("db cannot be nil"))
	}
	if c.rawOutputRefRepository != nil {
		return c.rawOutputRefRepository
	}
	c.rawOutputRefRepository = &repository.RawOutputRefRepository{
		Db: c.db,
	}
	err := c.rawOutputRefRepository.CreateTable(ctx)
	if err != nil {
		panic(err)
	}
	return c.rawOutputRefRepository
}

func (c *Container) GetInputRepository(ctx context.Context) *repository.InputRepository {
	if c.inputRepository != nil {
		return c.inputRepository
	}
	c.inputRepository = &repository.InputRepository{
		Db: c.db,
	}
	err := c.inputRepository.CreateTables(ctx)
	if err != nil {
		panic(err)
	}
	return c.inputRepository
}

func (c *Container) GetReportRepository(ctx context.Context) *repository.ReportRepository {
	if c.reportRepository != nil {
		return c.reportRepository
	}
	c.reportRepository = &repository.ReportRepository{
		Db:        c.db,
		AutoCount: c.AutoCount,
	}
	err := c.reportRepository.CreateTables(ctx)
	if err != nil {
		panic(err)
	}
	return c.reportRepository
}

func (c *Container) GetConvenienceService(ctx context.Context) *services.ConvenienceService {
	if c.convenienceService != nil {
		return c.convenienceService
	}
	c.convenienceService = services.NewConvenienceService(
		c.GetVoucherRepository(ctx),
		c.GetNoticeRepository(ctx),
		c.GetInputRepository(ctx),
		c.GetReportRepository(ctx),
		c.GetApplicationRepository(ctx),
	)
	return c.convenienceService
}
