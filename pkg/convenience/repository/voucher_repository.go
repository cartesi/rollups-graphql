package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

const FALSE = "false"

type VoucherRepository struct {
	Db               sqlx.DB
	OutputRepository OutputRepository
	AutoCount        bool
}

type voucherRow struct {
	Destination          string `db:"destination"`
	Payload              string `db:"payload"`
	InputIndex           uint64 `db:"input_index"`
	OutputIndex          uint64 `db:"output_index"`
	Executed             bool   `db:"executed"`
	Value                string `db:"value"`
	OutputHashesSiblings string `db:"output_hashes_siblings"`
	AppContract          string `db:"app_contract"`
	TransactionHash      string `db:"transaction_hash"`
	ProofOutputIndex     uint64 `db:"proof_output_index"`
	IsDelegatedCall      bool   `db:"is_delegated_call"`
}

func (c *VoucherRepository) CreateTables() error {
	schema := `CREATE TABLE IF NOT EXISTS vouchers (
		destination            text,
		payload 	           text,
		executed	           BOOLEAN,
		input_index            integer,
		output_index           integer,
		value		           text,
		output_hashes_siblings text,
		app_contract           text,
		transaction_hash       text DEFAULT '' NOT NULL,
		proof_output_index     integer DEFAULT 0,
		is_delegated_call	   BOOLEAN,
		PRIMARY KEY (input_index, output_index, app_contract)
	);

	CREATE INDEX IF NOT EXISTS idx_input_index_output_index ON vouchers(input_index, output_index);
	CREATE INDEX IF NOT EXISTS idx_app_contract_output_index ON vouchers(app_contract, output_index);
	CREATE INDEX IF NOT EXISTS idx_app_contract_input_index ON vouchers(app_contract, input_index);
	`

	// execute a query on the server
	_, err := c.Db.Exec(schema)
	return err
}

func (c *VoucherRepository) CreateVoucher(
	ctx context.Context, voucher *model.ConvenienceVoucher,
) (*model.ConvenienceVoucher, error) {
	// slog.Debug("CreateVoucher", "payload", voucher.Payload, "value", voucher.Value)
	if c.AutoCount {
		count, err := c.OutputRepository.CountAllOutputs(ctx)
		if err != nil {
			return nil, err
		}
		voucher.OutputIndex = count
		voucher.ProofOutputIndex = count
	}
	insertVoucher := `INSERT INTO vouchers (
		destination,
		payload,
		executed,
		input_index,
		output_index,
		value,
		output_hashes_siblings,
		app_contract,
		proof_output_index,
		is_delegated_call
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	exec := DBExecutor{&c.Db}

	_, err := exec.ExecContext(
		ctx,
		insertVoucher,
		voucher.Destination.Hex(),
		voucher.Payload,
		voucher.Executed,
		voucher.InputIndex,
		voucher.OutputIndex,
		voucher.Value,
		voucher.OutputHashesSiblings,
		voucher.AppContract.Hex(),
		voucher.ProofOutputIndex,
		voucher.IsDelegatedCall,
	)
	if err != nil {
		slog.Error("Error creating vouchers", "Error", err)
		return nil, err
	}
	return voucher, nil
}

func (c *VoucherRepository) SetProof(
	ctx context.Context, voucher *model.ConvenienceVoucher,
) error {
	updateVoucher := `UPDATE vouchers SET
		output_hashes_siblings = $1
		WHERE app_contract = $2 and output_index = $3`
	exec := DBExecutor{&c.Db}
	res, err := exec.ExecContext(
		ctx,
		updateVoucher,
		voucher.OutputHashesSiblings,
		voucher.AppContract.Hex(),
		voucher.OutputIndex,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("repo_err_1 wrong number of vouchers affected: %d; app_contract %v; output_index %d", affected, voucher.AppContract, voucher.OutputIndex)
	}
	return nil
}

func (c *VoucherRepository) SetExecuted(
	ctx context.Context, voucher *model.ConvenienceVoucher,
) error {
	updateVoucher := `UPDATE vouchers SET
		transaction_hash = $1,
		executed = true
		WHERE app_contract = $2 and output_index = $3`
	exec := DBExecutor{&c.Db}
	res, err := exec.ExecContext(
		ctx,
		updateVoucher,
		voucher.TransactionHash,
		voucher.AppContract.Hex(),
		voucher.OutputIndex,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("repo_err_2 wrong number of vouchers affected: %d; app_contract %v; output_index %d", affected, voucher.AppContract, voucher.OutputIndex)
	}
	return nil
}

func (c *VoucherRepository) UpdateVoucher(
	ctx context.Context, voucher *model.ConvenienceVoucher,
) (*model.ConvenienceVoucher, error) {
	updateVoucher := `UPDATE vouchers SET
		destination = $1,
		payload = $2,
		executed = $3
		WHERE input_index = $4 and output_index = $5`

	exec := DBExecutor{&c.Db}

	_, err := exec.ExecContext(
		ctx,
		updateVoucher,
		voucher.Destination.Hex(),
		voucher.Payload,
		voucher.Executed,
		voucher.InputIndex,
		voucher.OutputIndex,
	)
	if err != nil {
		return nil, err
	}

	return voucher, nil
}

func (c *VoucherRepository) DelegateCallVoucherCount(
	ctx context.Context,
) (uint64, error) {
	return c.voucherCount(ctx, true)
}

func (c *VoucherRepository) VoucherCount(
	ctx context.Context,
) (uint64, error) {
	return c.voucherCount(ctx, false)
}

func (c *VoucherRepository) voucherCount(
	ctx context.Context,
	isDelegatedCall bool,
) (uint64, error) {
	var count int
	err := c.Db.GetContext(ctx, &count, "SELECT count(*) FROM vouchers WHERE is_delegated_call = $1", isDelegatedCall)
	if err != nil {
		return 0, nil
	}
	return uint64(count), nil
}

func (c *VoucherRepository) queryByOutputIndexAndAppContract(
	ctx context.Context,
	outputIndex uint64,
	appContract *common.Address,
	isDelegatedCall bool,
) (*sqlx.Rows, error) {
	if appContract != nil {
		return c.Db.QueryxContext(ctx, `
			SELECT * FROM vouchers
			WHERE output_index = $1 and app_contract = $2 and is_delegated_call = $3
			LIMIT 1`,
			outputIndex,
			appContract.Hex(),
			isDelegatedCall,
		)
	} else {
		return c.Db.QueryxContext(ctx, `
			SELECT * FROM vouchers
			WHERE output_index = $1 and is_delegated_call = $2
			LIMIT 1`,
			outputIndex,
			isDelegatedCall,
		)
	}
}

func (c *VoucherRepository) FindVoucherByOutputIndexAndAppContract(
	ctx context.Context, outputIndex uint64,
	appContract *common.Address,
	isDelegatedCall bool,
) (*model.ConvenienceVoucher, error) {
	rows, err := c.queryByOutputIndexAndAppContract(ctx, outputIndex, appContract, isDelegatedCall)

	if err != nil {
		slog.Error("RollupsGraphql: FindVoucherByOutputIndexAndAppContract database error", "err", err)
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var row voucherRow
		if err := rows.StructScan(&row); err != nil {
			return nil, err
		}
		cVoucher := convertToConvenienceVoucher(row)

		return &cVoucher, nil
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return nil, nil
}

func (c *VoucherRepository) FindAllVouchersByBlockNumber(
	ctx context.Context, startBlockGte uint64, endBlockLt uint64, isDelegateCall bool,
) ([]*model.ConvenienceVoucher, error) {
	stmt, err := c.Db.Preparex(`
		SELECT
			v.destination,
			v.payload,
			v.executed,
			v.input_index,
			v.output_index,
			v.value,
			v.output_hashes_siblings,
			v.app_contract,
			v.is_delegated_call
		FROM vouchers v
			INNER JOIN convenience_inputs i
				ON i.app_contract = v.app_contract AND i.input_index = v.input_index
		WHERE i.block_number >= $1 and i.block_number < $2 and v.is_delegated_call = $3`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	var rows []voucherRow
	err = stmt.SelectContext(ctx, &rows, startBlockGte, endBlockLt, isDelegateCall)
	if err != nil {
		return nil, err
	}
	vouchers := make([]*model.ConvenienceVoucher, len(rows))
	for i, row := range rows {
		cVoucher := convertToConvenienceVoucher(row)
		vouchers[i] = &cVoucher
	}
	return vouchers, nil
}

func (c *VoucherRepository) FindVoucherByInputAndOutputIndex(
	ctx context.Context, inputIndex uint64, outputIndex uint64,
) (*model.ConvenienceVoucher, error) {

	query := `SELECT * FROM vouchers WHERE input_index = $1 and output_index = $2 LIMIT 1`

	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return nil, err
	}
	var row voucherRow
	err = stmt.GetContext(ctx, &row, inputIndex, outputIndex)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	defer stmt.Close()

	p := convertToConvenienceVoucher(row)

	return &p, nil
}

func (c *VoucherRepository) UpdateExecuted(
	ctx context.Context, inputIndex uint64, outputIndex uint64,
	executedValue bool,
) error {
	query := `UPDATE vouchers SET executed = $1 WHERE input_index = $2 and output_index = $3`
	_, err := c.Db.ExecContext(ctx, query, executedValue, inputIndex, outputIndex)
	if err != nil {
		return err
	}
	return nil
}

func (c *VoucherRepository) Count(
	ctx context.Context,
	filter []*model.ConvenienceFilter,
) (uint64, error) {
	return c.count(ctx, filter, false)
}

func (c *VoucherRepository) CountDelegateCall(
	ctx context.Context,
	filter []*model.ConvenienceFilter,
) (uint64, error) {
	return c.count(ctx, filter, true)
}

func (c *VoucherRepository) count(
	ctx context.Context,
	filter []*model.ConvenienceFilter,
	isDelegateCall bool,
) (uint64, error) {
	query := `SELECT count(*) FROM vouchers `
	filter = c.appendFilterDelegate(filter, isDelegateCall)
	where, args, _, err := transformToQuery(filter)
	if err != nil {
		return 0, err
	}
	query += where
	slog.Debug("Query", "query", query, "args", args)
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	var count uint64
	err = stmt.GetContext(ctx, &count, args...)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (c *VoucherRepository) FindAll(
	ctx context.Context,
) ([]model.ConvenienceVoucher, error) {
	query := `SELECT * FROM vouchers `
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	var rows []voucherRow
	err = stmt.SelectContext(ctx, &rows)
	if err != nil {
		return nil, err
	}

	vouchers := make([]model.ConvenienceVoucher, len(rows))

	for i, row := range rows {
		vouchers[i] = convertToConvenienceVoucher(row)
	}
	return vouchers, nil
}

func (c *VoucherRepository) FindAllVouchers(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	filter []*model.ConvenienceFilter,
) (*commons.PageResult[model.ConvenienceVoucher], error) {
	return c.findAllVouchers(ctx, first, last, after, before, filter, false)
}

func (c *VoucherRepository) FindAllDelegateCalls(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	filter []*model.ConvenienceFilter,
) (*commons.PageResult[model.ConvenienceVoucher], error) {
	return c.findAllVouchers(ctx, first, last, after, before, filter, true)
}

func (c *VoucherRepository) appendFilterDelegate(filter []*model.ConvenienceFilter, isDelegateCall bool) []*model.ConvenienceFilter {
	var (
		value  = strconv.FormatBool(isDelegateCall)
		key    = model.DELEGATED_CALL_VOUCHER
		output []*model.ConvenienceFilter
	)

	output = append(filter, &model.ConvenienceFilter{
		Field: &key,
		Eq:    &value,
	})

	return output
}

func (c *VoucherRepository) findAllVouchers(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	filter []*model.ConvenienceFilter,
	isDelegateCall bool,
) (*commons.PageResult[model.ConvenienceVoucher], error) {
	total, err := c.count(ctx, filter, isDelegateCall)
	if err != nil {
		return nil, err
	}
	query := `SELECT * FROM vouchers `

	filter = c.appendFilterDelegate(filter, isDelegateCall)
	where, args, argsCount, err := transformToQuery(filter)
	if err != nil {
		return nil, err
	}
	query += where

	query += ` ORDER BY input_index ASC, output_index ASC `
	offset, limit, err := commons.ComputePage(first, last, after, before, int(total))

	if err != nil {
		return nil, err
	}

	query += `LIMIT $` + strconv.Itoa(argsCount) + ` `
	args = append(args, limit)
	argsCount = argsCount + 1
	query += `OFFSET $` + strconv.Itoa(argsCount) + ` `
	args = append(args, offset)

	slog.Debug("Query", "query", query, "args", args, "total", total)
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	var rows []voucherRow
	err = stmt.SelectContext(ctx, &rows, args...)
	if err != nil {
		return nil, err
	}

	vouchers := make([]model.ConvenienceVoucher, len(rows))

	for i, row := range rows {
		vouchers[i] = convertToConvenienceVoucher(row)
	}

	pageResult := &commons.PageResult[model.ConvenienceVoucher]{
		Rows:   vouchers,
		Total:  total,
		Offset: uint64(offset),
	}
	return pageResult, nil
}

func (c *VoucherRepository) FindVoucherByAppContractAndIndex(ctx context.Context, index int, appContract common.Address) (*model.ConvenienceVoucher, error) {

	query := `SELECT * FROM convenience_vouchers WHERE input_index = $1 AND app_contract = $2`

	res, err := c.Db.QueryxContext(
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
		voucher, err := parseVoucher(res)
		if err != nil {
			return nil, err
		}
		return voucher, nil
	}
	return nil, nil
}

func convertToConvenienceVoucher(row voucherRow) model.ConvenienceVoucher {
	destinationAddress := common.HexToAddress(row.Destination)
	appContract := common.HexToAddress(row.AppContract)
	voucher := model.ConvenienceVoucher{
		Destination:          destinationAddress,
		Payload:              row.Payload,
		InputIndex:           row.InputIndex,
		OutputIndex:          row.OutputIndex,
		Executed:             row.Executed,
		Value:                row.Value,
		AppContract:          appContract,
		OutputHashesSiblings: row.OutputHashesSiblings,
		TransactionHash:      row.TransactionHash,
		ProofOutputIndex:     row.ProofOutputIndex,
		IsDelegatedCall:      row.IsDelegatedCall,
	}
	return voucher
}

func transformToQuery(
	filter []*model.ConvenienceFilter,
) (string, []interface{}, int, error) {
	query := ""
	if len(filter) > 0 {
		query += "WHERE "
	}
	args := []interface{}{}
	where := []string{}

	count := 1
	for _, filter := range filter {
		if *filter.Field == model.DELEGATED_CALL_VOUCHER {
			if *filter.Eq == "true" {
				where = append(where, fmt.Sprintf("is_delegated_call = $%d ", count))
				args = append(args, true)
				count += 1
			} else if *filter.Eq == FALSE {
				where = append(where, fmt.Sprintf("is_delegated_call = $%d ", count))
				args = append(args, false)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf(
					"unexpected delegated call value %s", *filter.Eq,
				)
			}
		} else if *filter.Field == model.EXECUTED {
			if *filter.Eq == "true" {
				where = append(where, fmt.Sprintf("executed = $%d ", count))
				args = append(args, true)
				count += 1
			} else if *filter.Eq == FALSE {
				where = append(where, fmt.Sprintf("executed = $%d ", count))
				args = append(args, false)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf(
					"unexpected executed value %s", *filter.Eq,
				)
			}
		} else if *filter.Field == model.DESTINATION {
			if filter.Eq != nil {
				where = append(where, fmt.Sprintf("destination = $%d ", count))
				if !common.IsHexAddress(*filter.Eq) {
					return "", nil, 0, fmt.Errorf("wrong address value")
				}
				args = append(args, *filter.Eq)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented")
			}
		} else if *filter.Field == model.INPUT_INDEX {
			if filter.Eq != nil {
				where = append(where, fmt.Sprintf("input_index = $%d ", count))
				args = append(args, *filter.Eq)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented")
			}
		} else if *filter.Field == model.APP_CONTRACT {
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

func (c *VoucherRepository) BatchFindAllByInputIndexAndAppContract(
	ctx context.Context,
	filters []*BatchFilterItem,
) ([]*commons.PageResult[model.ConvenienceVoucher], []error) {
	slog.Debug("BatchFindAllByInputIndexAndAppContract", "len", len(filters))
	query := `SELECT * FROM vouchers WHERE `

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
	results := []*commons.PageResult[model.ConvenienceVoucher]{}
	stmt, err := c.Db.PreparexContext(ctx, query)
	if err != nil {
		slog.Error("BatchFind prepare context", "error", err)
		return nil, errors
	}
	defer stmt.Close()

	rows, err := stmt.QueryxContext(ctx, args...)
	if err != nil {
		slog.Error("BatchFind query context", "error", err)
		return nil, errors
	}
	defer rows.Close()

	var voucherRows []voucherRow
	err = stmt.SelectContext(ctx, &voucherRows, args...)
	if err != nil {
		return nil, errors
	}

	vouchers := make([]model.ConvenienceVoucher, len(voucherRows))

	for i, row := range voucherRows {
		vouchers[i] = convertToConvenienceVoucher(row)
	}

	if err := rows.Err(); err != nil {
		return nil, errors
	}

	voucherMap := make(map[string]*commons.PageResult[model.ConvenienceVoucher])

	for _, voucher := range vouchers {
		key := GenerateBatchVoucherKey(&voucher.AppContract, int(voucher.InputIndex))
		if voucherMap[key] == nil {
			voucherMap[key] = &commons.PageResult[model.ConvenienceVoucher]{}
		}
		voucherMap[key].Total += 1
		voucherMap[key].Rows = append(voucherMap[key].Rows, voucher)
	}

	for _, filter := range filters {
		key := GenerateBatchVoucherKey(filter.AppContract, filter.InputIndex)
		reportsItem := voucherMap[key]
		if reportsItem == nil {
			reportsItem = &commons.PageResult[model.ConvenienceVoucher]{}
		}
		results = append(results, reportsItem)
	}
	slog.Debug("BatchVouchersResult", "len", len(results))
	return results, nil
}

func GenerateBatchVoucherKey(appContract *common.Address, inputIndex int) string {
	return fmt.Sprintf("%s|%d", appContract.Hex(), inputIndex)
}

func parseVoucher(res *sqlx.Rows) (*model.ConvenienceVoucher, error) {
	var (
		voucher     model.ConvenienceVoucher
		appContract string
		destination string
	)
	err := res.Scan(
		&destination,
		&voucher.Payload,
		&voucher.Executed,
		&voucher.InputIndex,
		&voucher.OutputIndex,
		&appContract,
	)
	if err != nil {
		return nil, err
	}

	voucher.AppContract = common.HexToAddress(appContract)
	voucher.Destination = common.HexToAddress(destination)
	return &voucher, nil
}
