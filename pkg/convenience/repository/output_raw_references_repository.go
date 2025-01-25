package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
)

const RAW_VOUCHER_TYPE = "voucher"
const RAW_NOTICE_TYPE = "notice"

type RawOutputRefRepository struct {
	Db *sqlx.DB
}

type RawOutputRef struct {
	AppID        uint64    `db:"app_id"`
	OutputIndex  uint64    `db:"output_index"`
	InputIndex   uint64    `db:"input_index"`
	AppContract  string    `db:"app_contract"`
	Type         string    `db:"type"`
	HasProof     bool      `db:"has_proof"`
	Executed     bool      `db:"executed"`
	UpdatedAt    time.Time `db:"updated_at"`
	CreatedAt    time.Time `db:"created_at"`
	SyncPriority uint64    `db:"sync_priority"`
}

func (r *RawOutputRefRepository) CreateTable() error {
	schema := `CREATE TABLE IF NOT EXISTS convenience_output_raw_references (
		app_id 			integer NOT NULL,
		input_index		integer NOT NULL,
		app_contract    text NOT NULL,
		output_index	integer NOT NULL,
		has_proof		BOOLEAN,
		type            text NOT NULL CHECK (type IN ('voucher', 'notice')),
		executed        BOOLEAN,
		updated_at      TIMESTAMP NOT NULL,
		created_at      TIMESTAMP NOT NULL,
		sync_priority   integer   NOT NULL,
		PRIMARY KEY (input_index, output_index, app_contract));
		
		CREATE INDEX IF NOT EXISTS idx_input_index ON convenience_output_raw_references(input_index, app_contract);
		CREATE INDEX IF NOT EXISTS idx_convenience_output_raw_references_app_id ON convenience_output_raw_references(app_id);
		CREATE INDEX IF NOT EXISTS idx_convenience_output_raw_references_has_proof_app_id ON convenience_output_raw_references(has_proof, app_id);`
	_, err := r.Db.Exec(schema)
	if err != nil {
		slog.Error("Failed to create Raw Outputs table", "error", err)
	} else {
		slog.Debug("Raw Outputs table created successfully")
	}
	return err
}

func (r *RawOutputRefRepository) Create(ctx context.Context, rawOutput RawOutputRef) error {
	exec := DBExecutor{r.Db}

	_, err := exec.ExecContext(ctx, `INSERT INTO convenience_output_raw_references (
		input_index,
		app_contract,
		output_index,
		type,
		app_id,
		has_proof,
		executed,
		updated_at,
		created_at,
		sync_priority
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		rawOutput.InputIndex,
		rawOutput.AppContract,
		rawOutput.OutputIndex,
		rawOutput.Type,
		rawOutput.AppID,
		rawOutput.HasProof,
		rawOutput.Executed,
		rawOutput.UpdatedAt,
		rawOutput.CreatedAt,
		time.Now().Unix(),
	)

	if err != nil {
		slog.Error("Error creating output reference", "error", err,
			"AppID", rawOutput.AppID,
			"OutputIndex", rawOutput.OutputIndex,
			"InputIndex", rawOutput.InputIndex,
		)
		return err
	}

	return err
}

func (r *RawOutputRefRepository) GetLatestRawOutputRef(ctx context.Context) (*RawOutputRef, error) {
	var outputRef RawOutputRef
	err := r.Db.GetContext(ctx, &outputRef, `
		SELECT * FROM convenience_output_raw_references
		ORDER BY created_at DESC, output_index DESC, app_id DESC
		LIMIT 1
	`)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		slog.Error("Failed to retrieve the latest output ID", "error", err)
		return nil, err
	}
	return &outputRef, err
}

func (r *RawOutputRefRepository) SetSyncPriority(ctx context.Context, rawOutputRef *RawOutputRef) error {
	exec := DBExecutor{r.Db}

	result, err := exec.ExecContext(ctx, `
		UPDATE convenience_output_raw_references
		SET 
			sync_priority = $1
		WHERE app_id = $2 and output_index = $3`,
		rawOutputRef.SyncPriority, rawOutputRef.AppID, rawOutputRef.OutputIndex)

	if err != nil {
		slog.Error("Error updating output proof", "error", err)
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		slog.Error("Error fetching rows affected", "error", err)
		return err
	}

	if affected != 1 {
		slog.Error("update error", "app_id", rawOutputRef.AppID, "output_index", rawOutputRef.OutputIndex)
		return fmt.Errorf("repo_err_1 unexpected number of rows updated: %d", affected)
	}

	return nil
}

func (r *RawOutputRefRepository) SetHasProofToTrue(ctx context.Context, rawOutputRef *RawOutputRef) error {
	exec := DBExecutor{r.Db}

	result, err := exec.ExecContext(ctx, `
		UPDATE convenience_output_raw_references
		SET 
			has_proof = true,
			updated_at = $1,
			sync_priority = $2
		WHERE app_id = $3 and output_index = $4`, rawOutputRef.UpdatedAt,
		rawOutputRef.SyncPriority, rawOutputRef.AppID, rawOutputRef.OutputIndex)

	if err != nil {
		slog.Error("Error updating output proof", "error", err)
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		slog.Error("Error fetching rows affected", "error", err)
		return err
	}

	if affected != 1 {
		slog.Error("update error", "app_id", rawOutputRef.AppID, "output_index", rawOutputRef.OutputIndex)
		return fmt.Errorf("repo_err_1 unexpected number of rows updated: %d", affected)
	}

	return nil
}

func (r *RawOutputRefRepository) SetExecutedToTrue(ctx context.Context, rawOutputRef *RawOutputRef) error {
	exec := DBExecutor{r.Db}

	result, err := exec.ExecContext(ctx, `
		UPDATE convenience_output_raw_references
		SET executed = true, updated_at = $1
		WHERE app_id = $2 and output_index = $3
		`, rawOutputRef.UpdatedAt, rawOutputRef.AppID, rawOutputRef.OutputIndex)

	if err != nil {
		slog.Error("Error updating executed field", "error", err)
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		slog.Error("Error fetching rows affected", "error", err)
		return err
	}

	if affected != 1 {
		return fmt.Errorf("repo_err_2 unexpected number of rows updated: %d", affected)
	}
	slog.Debug("SetExecutedToTrue",
		"updated_at", rawOutputRef.UpdatedAt,
		"app_id", rawOutputRef.AppID,
		"output_index", rawOutputRef.OutputIndex,
	)
	return nil
}

func (r *RawOutputRefRepository) FindByAppIDAndOutputIndex(ctx context.Context, appID, outputIndex uint64) (*RawOutputRef, error) {
	var outputRef RawOutputRef
	err := r.Db.GetContext(ctx, &outputRef, `
		SELECT * FROM convenience_output_raw_references 
		WHERE app_id = $1 and output_index = $2`, appID, outputIndex)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Debug("Output reference not found",
				"app_id", appID,
				"output_index", outputIndex,
			)
			return nil, nil
		}
		slog.Error("Error finding output reference by ID", "error", err, "app_id", appID, "output_index", outputIndex)
		return nil, err
	}
	return &outputRef, nil
}

func (r *RawOutputRefRepository) UpdateSyncPriority(ctx context.Context, rawOutputRef *RawOutputRef) error {
	exec := DBExecutor{r.Db}

	result, err := exec.ExecContext(ctx, `
		UPDATE convenience_output_raw_references
		SET sync_priority = $1
		WHERE app_id = $2 AND output_index = $3
		`, time.Now().Unix(), rawOutputRef.AppID, rawOutputRef.OutputIndex)

	if err != nil {
		slog.Error("Error updating executed field", "error", err)
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		slog.Error("Error fetching rows affected", "error", err)
		return err
	}

	if affected != 1 {
		return fmt.Errorf("repo_err_3 unexpected number of rows updated: %d", affected)
	}
	slog.Debug("UpdateProofSyncAt",
		"updated_at", rawOutputRef.UpdatedAt,
		"app_id", rawOutputRef.AppID,
		"output_index", rawOutputRef.OutputIndex,
	)
	return nil
}

func (r *RawOutputRefRepository) GetFirstOutputRefWithoutProof(ctx context.Context) (*RawOutputRef, error) {
	var outputRef RawOutputRef
	err := r.Db.GetContext(ctx, &outputRef, `
		SELECT 
			* 
		FROM
			convenience_output_raw_references 
		WHERE
			has_proof = false
		ORDER BY
			sync_priority ASC, updated_at ASC, output_index ASC, app_id ASC
		LIMIT 1`)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Debug("No output ID without proof found")
			return nil, nil
		}
		slog.Error("Failed to retrieve output ID without proof", "error", err)
		return nil, err
	}
	return &outputRef, err
}

func (r *RawOutputRefRepository) GetLastUpdatedAtExecuted(ctx context.Context) (*RawOutputRef, error) {
	var outputRef RawOutputRef
	err := r.Db.GetContext(ctx, &outputRef, `
		SELECT 
			*
		FROM
			convenience_output_raw_references 
		WHERE
			executed = true and type = 'voucher'
		ORDER BY updated_at DESC, app_id DESC, output_index DESC LIMIT 1`)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Debug("No output ref executed = true")
			return nil, nil
		}
		slog.Error("Failed to retrieve output ID not executed", "error", err)
		return nil, err
	}
	return &outputRef, err
}
