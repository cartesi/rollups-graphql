package repository

import (
	"context"
	"log/slog"

	"github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/jmoiron/sqlx"
)

type ApplicationRepository struct {
	Db sqlx.DB
}

func (a *ApplicationRepository) CreateTables() error {
	schema := `CREATE TABLE IF NOT EXISTS convenience_application (
		id INTEGER NOT NULL,
		name text NOT NULL,
		application_address text NOT NULL,
		consensus_address text NOT NULL,
		template_hash text NOT NULL,
		template_uri text NOT NULL,
		epoch_length text NOT NULL,
		state text NOT NULL,
		reason text,
		last_processed_block BIGINT NOT NULL,
		last_claim_check_block BIGINT NOT NULL,
		last_output_check_block BIGINT NOT NULL,
		processed_inputs BIGINT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);
	CREATE INDEX IF NOT EXISTS convenience_application_id ON convenience_application (id);
	CREATE INDEX IF NOT EXISTS convenience_application_application_address ON convenience_application (application_address);
	CREATE INDEX IF NOT EXISTS convenience_application_name ON convenience_application (name);
	`

	_, err := a.Db.Exec(schema)
	if err == nil {
		slog.Debug("Application table created")
	} else {
		slog.Error("Create table error", "error", err)
	}
	return err
}

func (a *ApplicationRepository) Create(ctx context.Context, rawApp *model.ConvenienceApplication) (*model.ConvenienceApplication, error) {
	insertSql := `INSERT INTO convenience_application (
		id,
		name,
		application_address,
		consensus_address,
		template_hash,
		template_uri,
		epoch_length,
		state,
		reason,
		last_processed_block,
		last_claim_check_block,
		last_output_check_block,
		processed_inputs,
		created_at,
		updated_at
		) VALUES (
		 $1,
		 $2,
		 $3,
		 $4,
		 $5,
		 $6,
		 $7,
		 $8,
		 $9,
		 $10,
		 $11,
		 $12,
		 $13,
		 $14,
		 $15
		);`

	exec := DBExecutor{db: &a.Db}

	_, err := exec.ExecContext(ctx, insertSql,
		rawApp.ID,
		rawApp.Name,
		rawApp.ApplicationAddress,
		rawApp.ConsensusAddress,
		rawApp.TemplateHash,
		rawApp.TemplateURI,
		rawApp.EpochLength,
		rawApp.State,
		rawApp.Reason,
		rawApp.LastProcessedBlock,
		rawApp.LastClaimCheckBlock,
		rawApp.LastOutputCheckBlock,
		rawApp.ProcessedInputs,
		rawApp.CreatedAt,
		rawApp.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return rawApp, nil
}

func (a *ApplicationRepository) Count(ctx context.Context, filter []*model.ConvenienceFilter) (uint64, error) {
	query := `SELECT COUNT(*) FROM convenience_application `
	where, args, _, err := transformToNoticeQuery(filter)
	if err != nil {
		return 0, err
	}
	query += where
	slog.Debug("Query", "query", query, "args", args)
	stmt, err := a.Db.Preparex(query)
	if err != nil {
		slog.Error("query error")
		return 0, err
	}
	defer stmt.Close()
	var countApplication uint64
	err = stmt.GetContext(ctx, &countApplication)
	if err != nil {
		return 0, err
	}
	return countApplication, nil
}
