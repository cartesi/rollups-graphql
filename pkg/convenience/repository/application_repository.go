package repository

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

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
		application_address text NOT NULL
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
		application_address
		) VALUES (
		 $1,
		 $2,
		 $3
		);`

	exec := DBExecutor{db: &a.Db}

	_, err := exec.ExecContext(ctx, insertSql,
		rawApp.ID,
		rawApp.Name,
		rawApp.ApplicationAddress.Hex(),
	)

	if err != nil {
		return nil, err
	}

	return rawApp, nil
}

func transformToApplicationQuery(filter []*model.ConvenienceFilter) (string, []any, int, error) {
	query := ""
	if len(filter) > 0 {
		query += WHERE
	}
	args := []any{}
	where := []string{}
	count := 1
	for _, filter := range filter {
		if *filter.Field == model.APP_CONTRACT {
			if filter.Eq != nil {
				where = append(
					where,
					fmt.Sprintf("application_address = $%d ", count),
				)
				args = append(args, *filter.Eq)
				count++
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

func (a *ApplicationRepository) FindAll(ctx context.Context, filter []*model.ConvenienceFilter) ([]*model.ConvenienceApplication, error) {
	query := `SELECT * FROM convenience_application `
	where, args, _, err := transformToApplicationQuery(filter)
	if err != nil {
		return nil, err
	}
	query += where
	slog.Debug("Query", "query", query, "args", args)
	stmt, err := a.Db.PreparexContext(ctx, query)
	if err != nil {
		slog.Error("query error")
		return nil, err
	}
	defer stmt.Close()
	var applications []*model.ConvenienceApplication
	err = stmt.SelectContext(ctx, &applications, args...)
	if err != nil {
		return nil, err
	}
	return applications, nil
}

func (a *ApplicationRepository) Count(ctx context.Context, filter []*model.ConvenienceFilter) (uint64, error) {
	query := `SELECT COUNT(*) FROM convenience_application `
	where, args, _, err := transformToApplicationQuery(filter)
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
	err = stmt.GetContext(ctx, &countApplication, args...)
	if err != nil {
		return 0, err
	}
	return countApplication, nil
}

func (a *ApplicationRepository) Update(ctx context.Context, data *model.ConvenienceApplication) error {
	return fmt.Errorf("not implemented")
}
