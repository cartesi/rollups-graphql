package reader

import (
	"context"

	graphql "github.com/cartesi/rollups-graphql/pkg/reader/model"
)

type Adapter interface {
	GetReport(
		ctx context.Context,
		reportIndex int,
	) (*graphql.Report, error)

	GetReports(
		ctx context.Context,
		first *int, last *int, after *string, before *string, inputIndex *int,
	) (*graphql.ReportConnection, error)

	GetAllReportsByInputIndex(
		ctx context.Context,
		inputIndex *int,
	) (*graphql.ReportConnection, error)

	GetInputs(
		ctx context.Context,
		first *int, last *int, after *string, before *string, where *graphql.InputFilter,
	) (*graphql.InputConnection, error)

	GetInput(
		ctx context.Context,
		id string,
	) (*graphql.Input, error)

	GetInputByIndex(
		ctx context.Context,
		inputIndex int,
	) (*graphql.Input, error)

	GetInputByIndexAppContract(
		ctx context.Context,
		inputIndex int,
		appContract string,
	) (*graphql.Input, error)

	GetNotice(
		ctx context.Context,
		outputIndex int,
	) (*graphql.Notice, error)

	GetNotices(
		ctx context.Context,
		first *int, last *int, after *string, before *string, inputIndex *int,
	) (*graphql.NoticeConnection, error)

	GetVoucher(
		ctx context.Context,
		outputIndex int) (*graphql.Voucher, error)

	GetDelegateCallVoucher(
		ctx context.Context,
		outputIndex int) (*graphql.DelegateCallVoucher, error)

	GetVouchers(
		ctx context.Context,
		first *int, last *int, after *string, before *string, inputIndex *int,
		filter []*graphql.ConvenientFilter,
	) (*graphql.VoucherConnection, error)

	GetDelegateCallVouchers(
		ctx context.Context,
		first *int, last *int, after *string, before *string, inputIndex *int,
		filter []*graphql.ConvenientFilter,
	) (*graphql.DelegateCallVoucherConnection, error)

	GetAllVouchersByInputIndex(
		ctx context.Context,
		inputIndex *int,
	) (*graphql.VoucherConnection, error)

	GetAllDelegateCallVouchersByInputIndex(
		ctx context.Context,
		inputIndex *int,
	) (*graphql.DelegateCallVoucherConnection, error)

	GetAllNoticesByInputIndex(
		ctx context.Context,
		inputIndex *int,
	) (*graphql.Connection[*graphql.Notice], error)

	GetApplications(
		ctx context.Context,
		first *int, last *int, after *string, before *string, filter *graphql.AppFilter,
	) (*graphql.Connection[*graphql.Application], error)

	GetApplicationByAppContract(
		ctx context.Context,
		appContract string,
	) (*graphql.Application, error)

	GetAllApplications(
		ctx context.Context,
		where *graphql.AppFilter,
	) (*graphql.Connection[*graphql.Application], error)
}
