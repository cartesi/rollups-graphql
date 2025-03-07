package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

type RawInputRefRepository struct {
	Db sqlx.DB
}

type RawInputRef struct {
	// RawID    uint64 `db:"raw_id"`    // Low level ID deprecated
	AppID       uint64    `db:"app_id"` // Low level app ID
	ID          string    `db:"id"`     // High level ID refers to our ConvenienceInput.ID
	InputIndex  uint64    `db:"input_index"`
	AppContract string    `db:"app_contract"`
	Status      string    `db:"status"`
	ChainID     string    `db:"chain_id"`
	CreatedAt   time.Time `db:"created_at"`
}

func (r *RawInputRefRepository) CreateTables() error {
	schema := `CREATE TABLE IF NOT EXISTS convenience_input_raw_references (
		id 				text NOT NULL,
		app_id 			integer NOT NULL,
		input_index		integer NOT NULL,
		app_contract    text NOT NULL,
		status	 		text,
		chain_id        text,
		created_at		TIMESTAMP NOT NULL);
	CREATE INDEX IF NOT EXISTS idx_input_index ON convenience_input_raw_references(input_index, app_contract);
	CREATE INDEX IF NOT EXISTS idx_app_id_input_index ON convenience_input_raw_references(app_id, input_index);
	CREATE INDEX IF NOT EXISTS idx_convenience_input_raw_references_status_raw_id ON convenience_input_raw_references(status, app_id);
	CREATE INDEX IF NOT EXISTS idx_status ON convenience_input_raw_references(status);`

	_, err := r.Db.Exec(schema)
	if err != nil {
		slog.Error("Failed to create tables", "error", err)
		return err
	}
	slog.Debug("Raw Inputs table created successfully")
	return nil
}

func (r *RawInputRefRepository) UpdateStatus(ctx context.Context, rawInputsRefs []RawInputRef, status string) error {
	if len(rawInputsRefs) == 0 {
		return nil
	}
	exec := DBExecutor{&r.Db}
	query := `UPDATE convenience_input_raw_references SET status = $1 WHERE `
	whereClauses := make([]string, len(rawInputsRefs))
	args := []interface{}{status}
	for i, input := range rawInputsRefs {
		// nolint
		placeholderIndex := 2 + i*2
		whereClauses[i] = fmt.Sprintf("(app_id = $%d AND input_index = $%d)", placeholderIndex, placeholderIndex+1)
		args = append(args, input.AppID, input.InputIndex)
	}
	query += strings.Join(whereClauses, " OR ")
	_, err := exec.ExecContext(ctx, query, args...)
	return err
}

func (r *RawInputRefRepository) Create(ctx context.Context, rawInput RawInputRef) error {
	exec := DBExecutor{&r.Db}

	appContract := common.HexToAddress(rawInput.AppContract)
	exist, err := r.FindByInputIndexAndAppContract(ctx, rawInput.InputIndex, &appContract)
	if err != nil {
		return err
	}
	if exist != nil {
		slog.Debug("Raw input already exists. Skipping creation")
		return nil
	}

	_, err = exec.ExecContext(ctx, `INSERT INTO convenience_input_raw_references (
		id, app_id, input_index, app_contract, status, chain_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		rawInput.ID, rawInput.AppID, rawInput.InputIndex,
		rawInput.AppContract, rawInput.Status, rawInput.ChainID,
		rawInput.CreatedAt,
	)

	if err != nil {
		slog.Error("Failed to insert raw input reference", "rawInput", rawInput, "error", err)
		return err
	}

	// slog.Debug("Raw input reference created", "ID", rawInput.ID)
	return nil
}

func (r *RawInputRefRepository) GetLatestInputRef(ctx context.Context) (*RawInputRef, error) {
	var inputRef RawInputRef
	err := r.Db.GetContext(ctx, &inputRef, `
		SELECT * FROM convenience_input_raw_references
		ORDER BY
			created_at DESC, input_index DESC, app_id DESC
		LIMIT 1
	`)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Warn("No raw input references found")
			return nil, nil
		}
		slog.Error("Failed to get latest raw input ref", "err", err)
		return nil, err
	}

	slog.Debug("Latest InputRef fetched",
		"app_id", inputRef.AppID,
		"input_index", inputRef.InputIndex,
	)
	return &inputRef, nil
}

func (r *RawInputRefRepository) FindFirstInputByStatusNone(ctx context.Context) (*RawInputRef, error) {
	query := `SELECT * FROM convenience_input_raw_references
			  WHERE status = 'NONE'
			  ORDER BY created_at ASC, input_index ASC, app_id ASC LIMIT 1`

	stmt, err := r.Db.PreparexContext(ctx, query)
	if err != nil {
		slog.Error("Failed to prepare query for status NONE", "query", query, "error", err)
		return nil, err
	}
	defer stmt.Close()

	var row RawInputRef

	err = stmt.GetContext(ctx, &row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Debug("No input found with status NONE")
			return nil, nil
		}
		slog.Error("Failed to execute query for status NONE", "error", err)
		return nil, err
	}

	slog.Debug("First input with status NONE fetched",
		"app_id", row.AppID, "input_index", row.InputIndex)
	return &row, nil
}

func (r *RawInputRefRepository) FindByInputIndexAndAppContract(ctx context.Context, inputIndex uint64, appContract *common.Address) (*RawInputRef, error) {
	var inputRef RawInputRef
	err := r.Db.GetContext(ctx, &inputRef, `
		SELECT * FROM convenience_input_raw_references
		WHERE input_index = $1 and app_contract = $2
		LIMIT 1`, inputIndex, appContract.Hex())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// slog.Debug("Input reference not found", "input_index", inputIndex)
			return nil, nil
		}
		slog.Error("Error finding input reference by input_index",
			"error", err,
			"input_index", inputIndex,
			"app_contract", appContract.Hex(),
		)
		return nil, err
	}
	return &inputRef, nil
}
