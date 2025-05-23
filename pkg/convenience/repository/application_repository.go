package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/cartesi/rollups-graphql/v2/pkg/commons"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

type ApplicationRepository struct {
	Db *sqlx.DB
}

func (a *ApplicationRepository) FindAppByAppContract(ctx context.Context, appContract *common.Address) (*model.ConvenienceApplication, error) {
	query := `SELECT id, name, app_contract FROM convenience_application WHERE app_contract = $1`
	stmt, err := a.Db.PreparexContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	var app model.ConvenienceApplication
	err = stmt.GetContext(ctx, &app, appContract.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &app, nil
}

func (a *ApplicationRepository) CreateTables(ctx context.Context) error {
	schema := `CREATE TABLE IF NOT EXISTS convenience_application (
		id INTEGER NOT NULL,
		name text NOT NULL,
		app_contract text NOT NULL
	);
	CREATE INDEX IF NOT EXISTS convenience_application_id ON convenience_application (id);
	CREATE INDEX IF NOT EXISTS convenience_application_app_contract ON convenience_application (app_contract);
	CREATE INDEX IF NOT EXISTS convenience_application_name ON convenience_application (name);
	`

	_, err := a.Db.ExecContext(ctx, schema)
	if err == nil {
		slog.DebugContext(ctx, "Application table created")
	} else {
		slog.ErrorContext(ctx, "Create table error", "error", err)
	}
	return err
}

func (a *ApplicationRepository) GetLatestApp(ctx context.Context) (*model.ConvenienceApplication, error) {
	query := `SELECT * FROM convenience_application ORDER BY id DESC LIMIT 1`
	stmt, err := a.Db.PreparexContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	var app model.ConvenienceApplication
	err = stmt.GetContext(ctx, &app)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}
	return &app, nil
}

func (a *ApplicationRepository) Create(ctx context.Context, rawApp *model.ConvenienceApplication) (*model.ConvenienceApplication, error) {
	insertSql := `INSERT INTO convenience_application (
		id,
		name,
		app_contract
		) VALUES (
		 $1,
		 $2,
		 $3
		);`

	exec := DBExecutor{db: a.Db}

	_, err := exec.ExecContext(ctx, insertSql,
		rawApp.ID,
		rawApp.Name,
		rawApp.ApplicationAddress,
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
					fmt.Sprintf("app_contract = $%d ", count),
				)
				args = append(args, *filter.Eq)
				count++
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented")
			}
		} else if *filter.Field == model.APP_NAME {
			if filter.Eq != nil {
				where = append(
					where,
					fmt.Sprintf("name = $%d ", count),
				)
				args = append(args, *filter.Eq)
				count++
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented")
			}
		} else if *filter.Field == model.APP_ID {
			if filter.Lt != nil {
				where = append(
					where,
					fmt.Sprintf("id < $%d ", count),
				)
				args = append(args, *filter.Lt)
				count++
			} else if filter.Gt != nil {
				where = append(
					where,
					fmt.Sprintf("id > $%d ", count),
				)
				args = append(args, *filter.Gt)
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

func (a *ApplicationRepository) FindAll(ctx context.Context, first *int, last *int, after *string, before *string, filter []*model.ConvenienceFilter) (*commons.PageResult[model.ConvenienceApplication], error) {
	total, err := a.Count(ctx, filter)
	if err != nil {
		return nil, err
	}
	where, args, argsCount, err := transformToApplicationQuery(filter)
	if err != nil {
		return nil, err
	}
	query := `SELECT * FROM convenience_application `
	query += where
	query += `ORDER BY id `

	offset, limit, err := commons.ComputePage(first, last, after, before, int(total))
	if err != nil {
		return nil, err
	}

	query += `LIMIT $` + strconv.Itoa(argsCount) + ` `
	args = append(args, limit)
	argsCount = argsCount + 1
	query += `OFFSET $` + strconv.Itoa(argsCount) + ` `
	args = append(args, offset)

	slog.DebugContext(ctx, "Query", "query", query, "args", args)
	stmt, err := a.Db.PreparexContext(ctx, query)
	if err != nil {
		slog.ErrorContext(ctx, "query error")
		return nil, err
	}
	defer stmt.Close()
	var applications []model.ConvenienceApplication
	err = stmt.SelectContext(ctx, &applications, args...)
	if err != nil {
		return nil, err
	}
	pagination := &commons.PageResult[model.ConvenienceApplication]{
		Rows:   applications,
		Total:  total,
		Offset: uint64(offset),
	}

	return pagination, nil
}

func (a *ApplicationRepository) Count(ctx context.Context, filter []*model.ConvenienceFilter) (uint64, error) {
	query := `SELECT COUNT(*) FROM convenience_application `
	where, args, _, err := transformToApplicationQuery(filter)
	if err != nil {
		return 0, err
	}
	query += where
	slog.DebugContext(ctx, "Query", "query", query, "args", args)
	stmt, err := a.Db.Preparex(query)
	if err != nil {
		slog.ErrorContext(ctx, "query error")
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
