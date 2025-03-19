package reader

import (
	"context"
	"fmt"
	"log/slog"

	cModel "github.com/cartesi/rollups-graphql/pkg/convenience/model"
	cRepos "github.com/cartesi/rollups-graphql/pkg/convenience/repository"
	services "github.com/cartesi/rollups-graphql/pkg/convenience/services"
	"github.com/cartesi/rollups-graphql/pkg/reader/loaders"
	graphql "github.com/cartesi/rollups-graphql/pkg/reader/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

type AdapterV1 struct {
	reportRepository   *cRepos.ReportRepository
	inputRepository    *cRepos.InputRepository
	voucherRepository  *cRepos.VoucherRepository
	convenienceService *services.ConvenienceService
}

// GetApplicationByAppContract implements Adapter.
func (a AdapterV1) GetApplicationByAppContract(ctx context.Context, appContract string) (*graphql.Application, error) {
	apps, err := a.GetAllApplications(ctx, &graphql.AppFilter{Address: &appContract})

	if err != nil {
		return nil, err
	}

	if len(apps.Edges) == 0 {
		return nil, fmt.Errorf("application not found")
	}

	app := apps.Edges[0].Node

	return app, nil
}

// GetAllApplications implements Adapter.
func (a AdapterV1) GetAllApplications(ctx context.Context, where *graphql.AppFilter) (*graphql.Connection[*graphql.Application], error) {
	return a.GetApplications(ctx, nil, nil, nil, nil, where)
}

func GetAppContractInsideCtx(ctx context.Context) (*common.Address, error) {
	appContractParam := ctx.Value(cModel.AppContractKey)
	if appContractParam == nil {
		return nil, fmt.Errorf("app contract not found in context")
	}
	appContract, ok := appContractParam.(string)
	if !ok {
		return nil, fmt.Errorf("invalid app contract type")
	}
	value := common.HexToAddress(appContract)
	return &value, nil
}

// GetApplicationByOutputIndex implements Adapter.
func (a AdapterV1) GetApplicationByOutputIndex(ctx context.Context, outputIndex uint64) (*graphql.Application, error) {
	panic("unimplemented")
	// var address *common.Address

	// // Trying to get app contract from context
	// ctxAddress, err := GetAppContractInsideCtx(ctx)
	// if err != nil {
	// 	slog.Debug("app contract not found in context", "error", err)
	// }

	// if ctxAddress != nil {
	// 	address = ctxAddress
	// } else {
	// 	// Trying to get app contract from inputBoxIndex
	// 	input, err := a.convenienceService.InputRepository.FindByIndexAndAppContract(ctx, outputIndex, nil)

	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	if input == nil {
	// 		slog.Debug("input not found", "inputBoxIndex", outputIndex)
	// 		return nil, fmt.Errorf("input from inputBoxIndex not found")
	// 	}
	// 	address = &input.AppContract
	// }

	// app, err := a.convenienceService.FindAppByAppContract(ctx, address)

	// if err != nil {
	// 	return nil, err
	// }

	// if app == nil {
	// 	slog.Debug("application not found", "appContract", address.Hex())
	// 	defaultApplication := &graphql.Application{
	// 		ID:      "0",
	// 		Name:    "MAIN",
	// 		Address: address.Hex(),
	// 	}
	// 	slog.Debug("Generate default application", "defaultApplication", defaultApplication)
	// 	return defaultApplication, nil
	// }

	// return graphql.ConvertToApplicationV1(*app), nil
}

// GetApplications implements Adapter.
func (a AdapterV1) GetApplications(ctx context.Context, first *int, last *int, after *string, before *string, filter *graphql.AppFilter) (*graphql.AppConnection, error) {
	filters, err := graphql.ConvertToAppFilter(filter)
	if err != nil {
		return nil, err
	}
	apps, err := a.convenienceService.FindAllApps(ctx, first, last, after, before, filters)
	if err != nil {
		return nil, err
	}
	return graphql.ConvertToAppConnectionV1(apps.Rows, int(apps.Offset), int(apps.Total))
}

func NewAdapterV1(
	db *sqlx.DB,
	convenienceService *services.ConvenienceService,
) Adapter {
	slog.Debug("NewAdapterV1")
	reportRepository := &cRepos.ReportRepository{
		Db: db,
	}
	err := reportRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	inputRepository := &cRepos.InputRepository{
		Db: *db,
	}
	err = inputRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	voucherRepository := &cRepos.VoucherRepository{
		Db: *db,
	}
	err = voucherRepository.CreateTables()
	if err != nil {
		panic(err)
	}

	return AdapterV1{
		reportRepository:   reportRepository,
		inputRepository:    inputRepository,
		voucherRepository:  voucherRepository,
		convenienceService: convenienceService,
	}
}

func (a AdapterV1) GetNotices(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	inputIndex *int,
) (*graphql.Connection[*graphql.Notice], error) {
	filters := []*cModel.ConvenienceFilter{}
	filters, err := addAppContractFilterAsNeeded(ctx, filters)
	if err != nil {
		return nil, err
	}
	if inputIndex != nil {
		field := cModel.INPUT_INDEX
		value := fmt.Sprintf("%d", *inputIndex)
		filters = append(filters, &cModel.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	notices, err := a.convenienceService.FindAllNotices(
		ctx,
		first,
		last,
		after,
		before,
		filters,
	)
	if err != nil {
		return nil, err
	}
	return graphql.ConvertToNoticeConnectionV1(
		notices.Rows,
		int(notices.Offset),
		int(notices.Total),
	)
}

func (a AdapterV1) GetVoucher(ctx context.Context, outputIndex int) (*graphql.Voucher, error) {
	appContract, err := getAppContractFromContext(ctx)
	if err != nil {
		return nil, err
	}
	voucher, err := a.convenienceService.FindVoucherByOutputIndexAndAppContract(
		ctx, uint64(outputIndex), appContract, false)
	if err != nil {
		return nil, err
	}
	if voucher == nil {
		return nil, fmt.Errorf("voucher not found")
	}
	return graphql.ConvertConvenientVoucherV1(*voucher), nil
}

// GetAllDelegateCallVouchersByInputIndex implements Adapter.
func (a AdapterV1) GetAllDelegateCallVouchersByInputIndex(ctx context.Context, inputIndex *int) (*graphql.DelegateCallVoucherConnection, error) {
	loaders := loaders.For(ctx)
	if loaders == nil {
		return a.GetDelegateCallVouchers(ctx, nil, nil, nil, nil, inputIndex, nil)
	} else {
		appContract, err := getAppContractFromContext(ctx)
		if err != nil {
			return nil, err
		}
		key := cRepos.GenerateBatchVoucherKey(appContract, *inputIndex)
		vouchers, err := loaders.VoucherLoader.Load(ctx, key)
		if err != nil {
			return nil, err
		}
		return graphql.ConvertToDelegateCallVoucherConnectionV1(
			vouchers.Rows,
			int(vouchers.Offset),
			int(vouchers.Total),
		)
	}
}

// GetDelegateCallVoucher implements Adapter.
func (a AdapterV1) GetDelegateCallVoucher(ctx context.Context, outputIndex int) (*graphql.DelegateCallVoucher, error) {
	appContract, err := getAppContractFromContext(ctx)
	if err != nil {
		return nil, err
	}
	voucher, err := a.convenienceService.FindVoucherByOutputIndexAndAppContract(
		ctx, uint64(outputIndex), appContract, true)
	if err != nil {
		return nil, err
	}
	if voucher == nil {
		return nil, fmt.Errorf("delegated call voucher not found")
	}
	return graphql.ConvertConvenientDelegateCallVoucherV1(*voucher), nil
}

// GetDelegateCallVouchers implements Adapter.
func (a AdapterV1) GetDelegateCallVouchers(ctx context.Context, first *int, last *int, after *string, before *string, inputIndex *int, filter []*graphql.ConvenientFilter) (*graphql.DelegateCallVoucherConnection, error) {
	filters, err := graphql.ConvertToConvenienceFilter(filter)
	if err != nil {
		return nil, err
	}
	filters, err = addAppContractFilterAsNeeded(ctx, filters)
	if err != nil {
		return nil, err
	}
	if inputIndex != nil {
		field := cModel.INPUT_INDEX
		value := fmt.Sprintf("%d", *inputIndex)
		filters = append(filters, &cModel.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	vouchers, err := a.convenienceService.FindAllDelegateCalls(
		ctx,
		first,
		last,
		after,
		before,
		filters,
	)
	if err != nil {
		return nil, err
	}
	return graphql.ConvertToDelegateCallVoucherConnectionV1(
		vouchers.Rows,
		int(vouchers.Offset),
		int(vouchers.Total),
	)
}

func getAppContractFromContext(ctx context.Context) (*common.Address, error) {
	appContractParam := ctx.Value(cModel.AppContractKey)
	if appContractParam != nil {
		appContract, ok := appContractParam.(string)
		if !ok {
			return nil, fmt.Errorf("wrong app contract type")
		}
		value := common.HexToAddress(appContract)
		return &value, nil
	}
	return nil, nil
}

func addAppContractFilterAsNeeded(ctx context.Context, filters []*cModel.ConvenienceFilter) ([]*cModel.ConvenienceFilter, error) {
	appContract, err := getAppContractFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if appContract != nil {
		field := cModel.APP_CONTRACT
		value := appContract.Hex()
		filters = append(filters, &cModel.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	return filters, nil
}

func (a AdapterV1) GetVouchers(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	inputIndex *int,
	filter []*graphql.ConvenientFilter,
) (*graphql.Connection[*graphql.Voucher], error) {
	filters, err := graphql.ConvertToConvenienceFilter(filter)
	if err != nil {
		return nil, err
	}
	filters, err = addAppContractFilterAsNeeded(ctx, filters)
	if err != nil {
		return nil, err
	}
	if inputIndex != nil {
		field := cModel.INPUT_INDEX
		value := fmt.Sprintf("%d", *inputIndex)
		filters = append(filters, &cModel.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	vouchers, err := a.convenienceService.FindAllVouchers(
		ctx,
		first,
		last,
		after,
		before,
		filters,
	)
	if err != nil {
		return nil, err
	}
	return graphql.ConvertToVoucherConnectionV1(
		vouchers.Rows,
		int(vouchers.Offset),
		int(vouchers.Total),
	)
}

func (a AdapterV1) GetAllNoticesByInputIndex(ctx context.Context, inputIndex *int) (*graphql.Connection[*graphql.Notice], error) {
	loaders := loaders.For(ctx)
	if loaders == nil {
		return a.GetNotices(ctx, nil, nil, nil, nil, inputIndex)
	} else {
		appContract, err := getAppContractFromContext(ctx)
		if err != nil {
			return nil, err
		}
		key := cRepos.GenerateBatchNoticeKey(appContract.Hex(), uint64(*inputIndex))
		notices, err := loaders.NoticeLoader.Load(ctx, key)
		if err != nil {
			return nil, err
		}
		return graphql.ConvertToNoticeConnectionV1(
			notices.Rows,
			int(notices.Offset),
			int(notices.Total),
		)
	}
}

func (a AdapterV1) GetAllVouchersByInputIndex(ctx context.Context, inputIndex *int) (*graphql.Connection[*graphql.Voucher], error) {
	loaders := loaders.For(ctx)
	if loaders == nil {
		return a.GetVouchers(ctx, nil, nil, nil, nil, inputIndex, nil)
	} else {
		appContract, err := getAppContractFromContext(ctx)
		if err != nil {
			return nil, err
		}
		key := cRepos.GenerateBatchVoucherKey(appContract, *inputIndex)
		vouchers, err := loaders.VoucherLoader.Load(ctx, key)
		if err != nil {
			return nil, err
		}
		return graphql.ConvertToVoucherConnectionV1(
			vouchers.Rows,
			int(vouchers.Offset),
			int(vouchers.Total),
		)
	}
}

func (a AdapterV1) GetNotice(ctx context.Context, outputIndex int) (*graphql.Notice, error) {
	appContract, err := getAppContractFromContext(ctx)
	if err != nil {
		return nil, err
	}
	notice, err := a.convenienceService.FindNoticeByOutputIndexAndAppContract(
		ctx,
		uint64(outputIndex),
		appContract,
	)
	if err != nil {
		return nil, err
	}
	if notice == nil {
		return nil, fmt.Errorf("notice not found")
	}
	return graphql.ConvertConvenientNoticeV1(*notice), nil
}

func (a AdapterV1) GetReport(
	ctx context.Context,
	reportIndex int,
) (*graphql.Report, error) {
	appContract, err := getAppContractFromContext(ctx)
	if err != nil {
		return nil, err
	}
	report, err := a.reportRepository.FindByOutputIndexAndAppContract(
		ctx,
		uint64(reportIndex),
		appContract,
	)
	if err != nil {
		return nil, err
	}
	if report == nil {
		return nil, fmt.Errorf("report not found")
	}
	return a.convertToReport(*report), nil
}

func (a AdapterV1) GetReports(
	ctx context.Context,
	first *int, last *int, after *string, before *string, inputIndex *int,
) (*graphql.ReportConnection, error) {
	filters, err := graphql.ConvertToConvenienceFilter(nil)
	if err != nil {
		return nil, err
	}
	filters, err = addAppContractFilterAsNeeded(ctx, filters)
	if err != nil {
		return nil, err
	}
	if inputIndex != nil {
		field := cModel.INPUT_INDEX
		value := fmt.Sprintf("%d", *inputIndex)
		filters = append(filters, &cModel.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	reports, err := a.reportRepository.FindAll(
		ctx,
		first, last, after, before, filters,
	)
	if err != nil {
		slog.Error("Adapter GetReports", "error", err)
		return nil, err
	}
	return a.convertToReportConnection(
		reports.Rows,
		int(reports.Offset),
		int(reports.Total),
	)
}

func (a AdapterV1) GetAllReportsByInputIndex(ctx context.Context, inputIndex *int) (*graphql.Connection[*graphql.Report], error) {
	loaders := loaders.For(ctx)
	if loaders == nil {
		return a.GetReports(ctx, nil, nil, nil, nil, inputIndex)
	} else {
		appContract, err := getAppContractFromContext(ctx)
		if err != nil {
			return nil, err
		}
		key := cRepos.GenerateBatchReportKey(appContract, *inputIndex)
		reports, err := loaders.ReportLoader.Load(ctx, key)
		if err != nil {
			return nil, err
		}
		return a.convertToReportConnection(
			reports.Rows,
			int(reports.Offset),
			int(reports.Total),
		)
	}
}

func (a AdapterV1) convertToReportConnection(
	reports []cModel.Report,
	offset int, total int,
) (*graphql.ReportConnection, error) {
	convNodes := make([]*graphql.Report, len(reports))
	for i := range reports {
		convNodes[i] = a.convertToReport(reports[i])
	}
	return graphql.NewConnection(offset, total, convNodes), nil
}

func (a AdapterV1) convertToReport(
	report cModel.Report,
) *graphql.Report {
	return &graphql.Report{
		Index:       report.Index,
		InputIndex:  report.InputIndex,
		Payload:     report.Payload,
		AppContract: report.AppContract.Hex(),
	}
}

// GetInputByOutputIndex implements Adapter.
func (a AdapterV1) GetInputByOutputIndex(ctx context.Context, outputIndex uint64) (*graphql.Input, error) {
	panic("unimplemented")
}

// GetInputByIndex implements Adapter.
func (a AdapterV1) GetInputByIndex(
	ctx context.Context,
	inputIndex int,
) (*graphql.Input, error) {
	appContract, err := getAppContractFromContext(ctx)
	if err != nil {
		return nil, err
	}
	loaders := loaders.For(ctx)
	if loaders != nil {
		key := cRepos.GenerateBatchInputKey(appContract.Hex(), uint64(inputIndex))
		input, err := loaders.InputLoader.Load(ctx, key)
		if err != nil {
			return nil, err
		}
		return getConvertedInputFromGraphql(input)
	} else {
		input, err := a.inputRepository.FindByIndexAndAppContract(ctx, inputIndex, appContract)
		if err != nil {
			return nil, err
		}
		return getConvertedInputFromGraphql(input)
	}
}

func getConvertedInputFromGraphql(input *cModel.AdvanceInput) (*graphql.Input, error) {
	if input == nil {
		return nil, fmt.Errorf("input not found")
	}
	convertedInput, err := graphql.ConvertInput(*input)

	if err != nil {
		return nil, err
	}

	return convertedInput, nil
}

func (a AdapterV1) GetInput(
	ctx context.Context,
	id string) (*graphql.Input, error) {
	appContract, err := getAppContractFromContext(ctx)
	if err != nil {
		return nil, err
	}
	input, err := a.inputRepository.FindByIDAndAppContract(ctx, id, appContract)
	if err != nil {
		return nil, err
	}
	return getConvertedInputFromGraphql(input)
}

func (a AdapterV1) GetInputs(
	ctx context.Context,
	first *int, last *int, after *string, before *string, where *graphql.InputFilter,
) (*graphql.InputConnection, error) {
	appContract := ctx.Value(cModel.AppContractKey)
	slog.Debug("GetInputs", "appContract", appContract)
	filters := []*cModel.ConvenienceFilter{}
	filters, err := addAppContractFilterAsNeeded(ctx, filters)
	if err != nil {
		return nil, err
	}
	if where != nil {
		field := "Index"
		if where.IndexGreaterThan != nil {
			value := fmt.Sprintf("%d", *where.IndexGreaterThan)
			filters = append(filters, &cModel.ConvenienceFilter{
				Field: &field,
				Gt:    &value,
			})
		}
		if where.IndexLowerThan != nil {
			value := fmt.Sprintf("%d", *where.IndexLowerThan)
			filters = append(filters, &cModel.ConvenienceFilter{
				Field: &field,
				Lt:    &value,
			})
		}
		if where.MsgSender != nil {
			msgSenderField := "MsgSender"
			filters = append(filters, &cModel.ConvenienceFilter{
				Field: &msgSenderField,
				Eq:    where.MsgSender,
			})
		}
		if where.Type != nil {
			typeField := "Type"
			filters = append(filters, &cModel.ConvenienceFilter{
				Field: &typeField,
				Eq:    where.Type,
			})
		}
	}
	inputs, err := a.inputRepository.FindAll(
		ctx, first, last, after, before, filters,
	)
	if err != nil {
		return nil, err
	}
	return a.convertToInputConnection(
		inputs.Rows,
		int(inputs.Offset),
		int(inputs.Total),
	)
}

func (a AdapterV1) convertToInputConnection(
	inputs []cModel.AdvanceInput,
	offset int, total int,
) (*graphql.InputConnection, error) {
	convNodes := make([]*graphql.Input, len(inputs))
	for i := range inputs {
		convertedInput, err := graphql.ConvertInput(inputs[i])

		if err != nil {
			return nil, err
		}

		convNodes[i] = convertedInput
	}
	return graphql.NewConnection(offset, total, convNodes), nil
}
