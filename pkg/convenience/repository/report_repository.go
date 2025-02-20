package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	cModel "github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

type ReportRepository struct {
	Db        *sqlx.DB
	AutoCount bool
}

func (r *ReportRepository) CreateTables() error {
	schema := `CREATE TABLE IF NOT EXISTS convenience_reports (
    output_index  integer,
    payload       text,
    input_index   integer,
    app_contract  text,
    app_id        integer,
    PRIMARY KEY (input_index, output_index, app_contract)
	);

	CREATE INDEX IF NOT EXISTS idx_input_index_output_index ON convenience_reports(input_index, output_index);
	CREATE INDEX IF NOT EXISTS idx_input_index_app_contract ON convenience_reports(input_index, app_contract);
	CREATE INDEX IF NOT EXISTS idx_output_index_app_contract ON convenience_reports(output_index, app_contract);`
	_, err := r.Db.Exec(schema)
	if err == nil {
		slog.Debug("Reports table created")
	} else {
		slog.Error("Create table error", "error", err)
	}
	return err
}

func (r *ReportRepository) CreateReport(ctx context.Context, report cModel.Report) (cModel.Report, error) {
	if r.AutoCount {
		count, err := r.Count(ctx, nil)
		if err != nil {
			slog.Error("database error", "err", err)
			return cModel.Report{}, err
		}
		report.Index = int(count)
	}
	insertSql := `INSERT INTO convenience_reports (
		output_index,
		payload,
		input_index,
		app_contract,
		app_id) VALUES ($1, $2, $3, $4, $5)`

	var hexPayload string
	if !strings.HasPrefix(report.Payload, "0x") {
		hexPayload = "0x" + report.Payload
	} else {
		hexPayload = report.Payload
	}

	exec := DBExecutor{r.Db}
	_, err := exec.ExecContext(
		ctx,
		insertSql,
		report.Index,
		hexPayload,
		report.InputIndex,
		report.AppContract.Hex(),
		report.AppID,
	)

	if err != nil {
		slog.Error("database error", "err", err)
		return cModel.Report{}, err
	}
	// slog.Debug("Report created",
	// 	"outputIndex", report.Index,
	// 	"inputIndex", report.InputIndex,
	// )
	return report, nil
}

func (r *ReportRepository) Update(ctx context.Context, report cModel.Report) (*cModel.Report, error) {
	sql := `UPDATE convenience_reports
		SET payload = $1
		WHERE input_index = $2 and output_index = $3 `

	exec := DBExecutor{r.Db}
	_, err := exec.ExecContext(
		ctx,
		sql,
		report.Payload,
		report.InputIndex,
		report.Index,
	)
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *ReportRepository) queryByOutputIndexAndAppContract(
	ctx context.Context,
	outputIndex uint64,
	appContract *common.Address,
) (*sqlx.Rows, error) {
	if appContract != nil {
		return r.Db.QueryxContext(ctx, `
			SELECT payload, input_index FROM convenience_reports
			WHERE output_index = $1 and app_contract = $2
			LIMIT 1`,
			outputIndex,
			appContract.Hex(),
		)
	} else {
		return r.Db.QueryxContext(ctx, `
			SELECT payload, input_index FROM convenience_reports
			WHERE output_index = $1
			LIMIT 1`,
			outputIndex,
		)
	}
}

func (r *ReportRepository) FindLastReport(ctx context.Context) (*cModel.FastReport, error) {
	var report cModel.FastReport
	err := r.Db.GetContext(ctx, &report, `
		SELECT * FROM convenience_reports
		ORDER BY
			output_index DESC,
			app_id DESC
		LIMIT 1`)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		slog.Error("Failed to retrieve the last report from the database", "error", err)
		return nil, err
	}
	return &report, err
}

func (r *ReportRepository) FindByInputAndOutputIndex(
	ctx context.Context,
	inputIndex uint64,
	outputIndex uint64,
) (*cModel.Report, error) {
	rows, err := r.Db.QueryxContext(ctx, `
			SELECT payload FROM convenience_reports
			WHERE input_index = $1 AND output_index = $2
			LIMIT 1`,
		inputIndex, outputIndex,
	)
	if err != nil {
		slog.Error("database error", "err", err)
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		report := &cModel.Report{
			InputIndex: int(inputIndex),
			Index:      int(outputIndex),
			Payload:    payload,
		}
		return report, nil
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return nil, nil
}

func (r *ReportRepository) FindByOutputIndexAndAppContract(
	ctx context.Context,
	outputIndex uint64,
	appContract *common.Address,
) (*cModel.Report, error) {
	rows, err := r.queryByOutputIndexAndAppContract(ctx, outputIndex, appContract)
	if err != nil {
		slog.Error("database error", "err", err)
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var payload string
		var inputIndex int
		if err := rows.Scan(&payload, &inputIndex); err != nil {
			return nil, err
		}
		report := &cModel.Report{
			InputIndex: inputIndex,
			Index:      int(outputIndex),
			Payload:    payload,
		}
		return report, nil
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return nil, nil
}

func (r *ReportRepository) FindReportByAppContractAndIndex(ctx context.Context, index int, appContract common.Address) (*cModel.Report, error) {

	query := `SELECT
		input_index,
		output_index,
		payload,
		app_contract FROM convenience_reports WHERE input_index = $1 AND app_contract = $2`

	res, err := r.Db.QueryxContext(
		ctx,
		query,
		uint64(index),
		appContract.Hex(),
	)

	if err != nil {
		return nil, err
	}
	defer res.Close()

	if res.Next() {
		report, err := parseReport(res)
		if err != nil {
			return nil, err
		}
		return report, nil
	}
	return nil, nil
}

func (c *ReportRepository) Count(
	ctx context.Context,
	filter []*cModel.ConvenienceFilter,
) (uint64, error) {
	query := `SELECT count(*) FROM convenience_reports `
	where, args, _, err := transformToReportQuery(filter)
	if err != nil {
		slog.Error("Count execution error")
		return 0, err
	}
	query += where
	slog.Debug("Query", "query", query, "args", args)
	stmt, err := c.Db.PreparexContext(ctx, query)
	if err != nil {
		slog.Error("Count execution error")
		return 0, err
	}
	defer stmt.Close()
	var count uint64
	err = stmt.Get(&count, args...)
	if err != nil {
		slog.Error("Count execution error")
		return 0, err
	}
	return count, nil
}

func (c *ReportRepository) FindAllByInputIndex(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	inputIndex *int,
) (*commons.PageResult[cModel.Report], error) {
	filter := []*cModel.ConvenienceFilter{}
	if inputIndex != nil {
		field := cModel.INPUT_INDEX
		value := fmt.Sprintf("%d", *inputIndex)
		filter = append(filter, &cModel.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	return c.FindAll(
		ctx,
		first,
		last,
		after,
		before,
		filter,
	)
}

func (c *ReportRepository) FindAll(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	filter []*cModel.ConvenienceFilter,
) (*commons.PageResult[cModel.Report], error) {
	total, err := c.Count(ctx, filter)
	if err != nil {
		slog.Error("database error", "err", err)
		return nil, err
	}

	query := `SELECT input_index, output_index, payload FROM convenience_reports `
	where, args, argsCount, err := transformToReportQuery(filter)
	if err != nil {
		slog.Error("database error", "err", err)
		return nil, err
	}
	query += where
	query += `ORDER BY input_index ASC, output_index ASC `

	offset, limit, err := commons.ComputePage(first, last, after, before, int(total))
	if err != nil {
		return nil, err
	}
	query += fmt.Sprintf(`LIMIT $%d `, argsCount)
	args = append(args, limit)
	argsCount += 1
	query += fmt.Sprintf(`OFFSET $%d `, argsCount)
	args = append(args, offset)

	slog.Debug("Query", "query", query, "args", args, "total", total)
	stmt, err := c.Db.PreparexContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var reports []cModel.Report
	rows, err := stmt.QueryxContext(ctx, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var payload string
		var inputIndex int
		var outputIndex int
		if err := rows.Scan(&inputIndex, &outputIndex, &payload); err != nil {
			return nil, err
		}
		report := &cModel.Report{
			InputIndex: inputIndex,
			Index:      outputIndex,
			Payload:    payload,
		}
		reports = append(reports, *report)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	pageResult := &commons.PageResult[cModel.Report]{
		Rows:   reports,
		Total:  total,
		Offset: uint64(offset),
	}
	return pageResult, nil
}

func transformToReportQuery(
	filter []*cModel.ConvenienceFilter,
) (string, []interface{}, int, error) {
	query := ""
	if len(filter) > 0 {
		query += WHERE
	}
	args := []interface{}{}
	where := []string{}
	count := 1
	for _, filter := range filter {
		if *filter.Field == "OutputIndex" {
			if filter.Eq != nil {
				where = append(where, fmt.Sprintf("output_index = $%d ", count))
				args = append(args, *filter.Eq)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented")
			}
		} else if *filter.Field == cModel.INPUT_INDEX {
			if filter.Eq != nil {
				where = append(where, fmt.Sprintf("input_index = $%d ", count))
				args = append(args, *filter.Eq)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented")
			}
		} else if *filter.Field == cModel.APP_CONTRACT {
			if filter.Eq != nil {
				where = append(where, fmt.Sprintf("app_contract = $%d ", count))
				args = append(args, *filter.Eq)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented")
			}
		} else {
			return "", nil, 0, fmt.Errorf("unexpected field %s", *filter.Field)
		}
	}
	query += strings.Join(where, " and ")
	return query, args, count, nil
}

type BatchFilterItem struct {
	AppContract *common.Address
	InputIndex  int
}

func (c *ReportRepository) BatchFindAllByInputIndexAndAppContract(
	ctx context.Context,
	filters []*BatchFilterItem,
) ([]*commons.PageResult[cModel.Report], []error) {
	slog.Debug("BatchFindAllByInputIndexAndAppContract", "len", len(filters))
	query := `SELECT
				input_index, output_index, payload, app_contract FROM convenience_reports
		WHERE
	`
	args := []interface{}{}
	where := []string{}
	for i, filter := range filters {
		// nolint
		where = append(where, fmt.Sprintf(" (app_contract = $%d and input_index = $%d) ", i*2+1, i*2+2))
		args = append(args, filter.AppContract.Hex())
		args = append(args, filter.InputIndex)
	}
	query += strings.Join(where, " or ")

	errors := []error{}
	results := []*commons.PageResult[cModel.Report]{}
	stmt, err := c.Db.PreparexContext(ctx, query)
	if err != nil {
		slog.Error("BatchFind prepare context", "error", err)
		return nil, errors
	}
	defer stmt.Close()

	var reports []cModel.Report
	rows, err := stmt.QueryxContext(ctx, args...)
	if err != nil {
		slog.Error("BatchFind query context", "error", err)
		return nil, errors
	}
	defer rows.Close()

	for rows.Next() {
		var payload string
		var inputIndex int
		var outputIndex int
		var appContract string
		if err := rows.Scan(&inputIndex, &outputIndex, &payload, &appContract); err != nil {
			return nil, errors
		}
		report := &cModel.Report{
			InputIndex:  inputIndex,
			Index:       outputIndex,
			Payload:     payload,
			AppContract: common.HexToAddress(appContract),
		}
		reports = append(reports, *report)
	}

	if err := rows.Err(); err != nil {
		return nil, errors
	}
	reportMap := make(map[string]*commons.PageResult[cModel.Report])
	for _, report := range reports {
		key := GenerateBatchReportKey(&report.AppContract, report.InputIndex)
		if reportMap[key] == nil {
			reportMap[key] = &commons.PageResult[cModel.Report]{}
		}
		reportMap[key].Total += 1
		reportMap[key].Rows = append(reportMap[key].Rows, report)
	}
	for _, filter := range filters {
		key := GenerateBatchReportKey(filter.AppContract, filter.InputIndex)
		reportsItem := reportMap[key]
		if reportsItem == nil {
			reportsItem = &commons.PageResult[cModel.Report]{}
		}
		results = append(results, reportsItem)
	}
	slog.Debug("BatchResult", "len", len(results))
	return results, nil
}

func GenerateBatchReportKey(appContract *common.Address, inputIndex int) string {
	return fmt.Sprintf("%s|%d", appContract.Hex(), inputIndex)
}

func parseReport(res *sqlx.Rows) (*cModel.Report, error) {
	var (
		report      cModel.Report
		payload     string
		appContract string
	)
	err := res.Scan(
		&report.InputIndex,
		&report.Index,
		&payload,
		&appContract,
	)
	if err != nil {
		return nil, err
	}

	report.Payload = payload
	report.AppContract = common.HexToAddress(appContract)
	return &report, nil
}
