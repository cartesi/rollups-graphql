package services

import (
	"context"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/cartesi/rollups-graphql/pkg/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
)

type ConvenienceService struct {
	VoucherRepository *repository.VoucherRepository
	NoticeRepository  *repository.NoticeRepository
	InputRepository   *repository.InputRepository
	ReportRepository  *repository.ReportRepository
}

func NewConvenienceService(
	voucherRepository *repository.VoucherRepository,
	noticeRepository *repository.NoticeRepository,
	inputRepository *repository.InputRepository,
	reportRepository *repository.ReportRepository,
) *ConvenienceService {
	return &ConvenienceService{
		VoucherRepository: voucherRepository,
		NoticeRepository:  noticeRepository,
		InputRepository:   inputRepository,
		ReportRepository:  reportRepository,
	}
}

func (s *ConvenienceService) CreateVoucher1(
	ctx context.Context,
	voucher *model.ConvenienceVoucher,
) (*model.ConvenienceVoucher, error) {
	return s.VoucherRepository.CreateVoucher(ctx, voucher)
}

func (s *ConvenienceService) CreateNotice(
	ctx context.Context,
	notice *model.ConvenienceNotice,
) (*model.ConvenienceNotice, error) {
	noticeInDb, err := s.NoticeRepository.FindByInputAndOutputIndex(
		ctx, notice.InputIndex, notice.OutputIndex,
	)
	if err != nil {
		return nil, err
	}

	if noticeInDb != nil {
		return s.NoticeRepository.Update(ctx, notice)
	}
	return s.NoticeRepository.Create(ctx, notice)
}

func (s *ConvenienceService) CreateVoucher(
	ctx context.Context,
	voucher *model.ConvenienceVoucher,
) (*model.ConvenienceVoucher, error) {

	voucherInDb, err := s.VoucherRepository.FindVoucherByInputAndOutputIndex(
		ctx, voucher.InputIndex,
		voucher.OutputIndex,
	)

	if err != nil {
		return nil, err
	}

	if voucherInDb != nil {
		return s.VoucherRepository.UpdateVoucher(ctx, voucher)
	}

	return s.VoucherRepository.CreateVoucher(ctx, voucher)
}

func (s *ConvenienceService) CreateInput(
	ctx context.Context,
	input *model.AdvanceInput,
) (*model.AdvanceInput, error) {

	inputInDb, err := s.InputRepository.FindByIDAndAppContract(ctx, input.ID, &input.AppContract)

	if err != nil {
		return nil, err
	}

	if inputInDb != nil {
		return s.InputRepository.Update(ctx, *input)
	}
	return s.InputRepository.Create(ctx, *input)
}

func (c *ConvenienceService) UpdateExecuted(
	ctx context.Context,
	inputIndex uint64,
	outputIndex uint64,
	executedValue bool,
) error {
	return c.VoucherRepository.UpdateExecuted(
		ctx,
		inputIndex,
		outputIndex,
		executedValue,
	)
}

func (c *ConvenienceService) FindAllDelegateCalls(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	filter []*model.ConvenienceFilter,
) (*commons.PageResult[model.ConvenienceVoucher], error) {
	return c.VoucherRepository.FindAllDelegateCalls(
		ctx,
		first,
		last,
		after,
		before,
		filter,
	)
}
func (c *ConvenienceService) FindAllVouchers(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	filter []*model.ConvenienceFilter,
) (*commons.PageResult[model.ConvenienceVoucher], error) {
	return c.VoucherRepository.FindAllVouchers(
		ctx,
		first,
		last,
		after,
		before,
		filter,
	)
}

func (c *ConvenienceService) FindAllNotices(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	filter []*model.ConvenienceFilter,
) (*commons.PageResult[model.ConvenienceNotice], error) {
	return c.NoticeRepository.FindAllNotices(
		ctx,
		first,
		last,
		after,
		before,
		filter,
	)
}

func (c *ConvenienceService) FindAllInputs(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	filter []*model.ConvenienceFilter,
) (*commons.PageResult[model.AdvanceInput], error) {
	return c.InputRepository.FindAll(
		ctx,
		first,
		last,
		after,
		before,
		filter,
	)
}

func (c *ConvenienceService) FindAllByInputIndex(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	inputIndex *int,
) (*commons.PageResult[model.Report], error) {
	return c.ReportRepository.FindAllByInputIndex(
		ctx,
		first,
		last,
		after,
		before,
		inputIndex,
	)
}

func (c *ConvenienceService) FindVoucherByOutputIndexAndAppContract(
	ctx context.Context, outputIndex uint64,
	appContract *common.Address,
	isDelegateCall bool,
) (*model.ConvenienceVoucher, error) {
	return c.VoucherRepository.FindVoucherByOutputIndexAndAppContract(
		ctx, outputIndex, appContract, isDelegateCall,
	)
}

func (c *ConvenienceService) FindVoucherByInputAndOutputIndex(
	ctx context.Context, inputIndex uint64, outputIndex uint64,
) (*model.ConvenienceVoucher, error) {
	return c.VoucherRepository.FindVoucherByInputAndOutputIndex(
		ctx, inputIndex, outputIndex,
	)
}

func (c *ConvenienceService) FindNoticeByOutputIndexAndAppContract(
	ctx context.Context, outputIndex uint64,
	appContract *common.Address,
) (*model.ConvenienceNotice, error) {
	return c.NoticeRepository.FindNoticeByOutputIndexAndAppContract(
		ctx, outputIndex, appContract,
	)
}

func (c *ConvenienceService) FindNoticeByInputAndOutputIndex(
	ctx context.Context, inputIndex uint64, outputIndex uint64,
) (*model.ConvenienceNotice, error) {
	return c.NoticeRepository.FindByInputAndOutputIndex(
		ctx, inputIndex, outputIndex,
	)
}
