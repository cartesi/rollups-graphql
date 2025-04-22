// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This package contains the nonodo run function.
// This is separate from the main package to facilitate testing.
package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"time"

	"github.com/cartesi/rollups-graphql/v2/pkg/contracts"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience/synchronizer"
	synchronizernode "github.com/cartesi/rollups-graphql/v2/pkg/convenience/synchronizer_node"
	"github.com/cartesi/rollups-graphql/v2/pkg/health"
	"github.com/cartesi/rollups-graphql/v2/pkg/reader"
	"github.com/cartesi/rollups-graphql/v2/pkg/supervisor"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/spf13/cast"
)

const (
	DefaultHttpPort           = 8080
	DefaultRollupsPort        = 5004
	DefaultNamespace          = 10008
	DefaultMaxOpenConnections = 25
	DefaultMaxIdleConnections = 10
	DefaultConnMaxLifetime    = 30 * time.Minute
	DefaultConnMaxIdleTime    = 5 * time.Minute
)

// Options to nonodo.
type BootstrapOpts struct {
	AutoCount          bool
	HttpAddress        string
	HttpPort           int
	ApplicationAddress string
	SqliteFile         string
	DbImplementation   string
	TimeoutWorker      time.Duration
	DisableSync        bool
}

// Create the options struct with default values.
func NewBootstrapOpts() BootstrapOpts {
	return BootstrapOpts{
		HttpAddress:        "0.0.0.0",
		HttpPort:           DefaultHttpPort,
		ApplicationAddress: "0x75135d8ADb7180640d29d822D9AD59E83E8695b2",
		SqliteFile:         "",
		DbImplementation:   "postgres",
		TimeoutWorker:      supervisor.DefaultSupervisorTimeout,
		AutoCount:          false,
		DisableSync:        false,
	}
}

func NewSupervisorGraphQL(ctx context.Context, opts BootstrapOpts) supervisor.SupervisorWorker {
	var w supervisor.SupervisorWorker
	w.Timeout = opts.TimeoutWorker
	db := CreateDBInstance(ctx, opts)
	container := convenience.NewContainer(db, opts.AutoCount)
	convenienceService := container.GetConvenienceService(ctx)
	adapter := reader.NewAdapterV1(ctx, db, convenienceService)

	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Request timed out",
	}))
	health.Register(e)
	reader.Register(ctx, e, convenienceService, adapter)
	w.Workers = append(w.Workers, supervisor.HttpWorker{
		Address: fmt.Sprintf("%v:%v", opts.HttpAddress, opts.HttpPort),
		Handler: e,
	})

	if !opts.DisableSync {
		dbRawUrl, ok := os.LookupEnv("POSTGRES_NODE_DB_URL")
		if !ok {
			panic("POSTGRES_NODE_DB_URL environment variable not set")
		}
		dbNodeV2 := sqlx.MustConnect("postgres", dbRawUrl)
		rawRepository := synchronizernode.NewRawRepository(dbRawUrl, dbNodeV2)
		synchronizerUpdate := synchronizernode.NewSynchronizerUpdate(
			container.GetRawInputRepository(ctx),
			rawRepository,
			container.GetInputRepository(ctx),
		)
		synchronizerReport := synchronizernode.NewSynchronizerReport(
			container.GetReportRepository(ctx),
			rawRepository,
		)
		synchronizerOutputUpdate := synchronizernode.NewSynchronizerOutputUpdate(
			container.GetVoucherRepository(ctx),
			container.GetNoticeRepository(ctx),
			rawRepository,
			container.GetRawOutputRefRepository(ctx),
		)

		abi, err := contracts.OutputsMetaData.GetAbi()
		if err != nil {
			panic(err)
		}
		abiDecoder := synchronizernode.NewAbiDecoder(abi)

		inputAbi, err := contracts.InputsMetaData.GetAbi()
		if err != nil {
			panic(err)
		}

		inputAbiDecoder := synchronizernode.NewAbiDecoder(inputAbi)

		synchronizerOutputCreate := synchronizernode.NewSynchronizerOutputCreate(
			container.GetVoucherRepository(ctx),
			container.GetNoticeRepository(ctx),
			rawRepository,
			container.GetRawOutputRefRepository(ctx),
			abiDecoder,
		)

		synchronizerOutputExecuted := synchronizernode.NewSynchronizerOutputExecuted(
			container.GetVoucherRepository(ctx),
			container.GetNoticeRepository(ctx),
			rawRepository,
			container.GetRawOutputRefRepository(ctx),
		)

		synchronizerInputCreate := synchronizernode.NewSynchronizerInputCreator(
			container.GetInputRepository(ctx),
			container.GetRawInputRepository(ctx),
			rawRepository,
			inputAbiDecoder,
		)

		synchronizerAppCreate := synchronizernode.NewSynchronizerAppCreator(container.GetApplicationRepository(ctx), rawRepository)

		synchronizerWorker := synchronizernode.NewSynchronizerCreateWorker(
			container.GetInputRepository(ctx),
			container.GetRawInputRepository(ctx),
			dbRawUrl,
			rawRepository,
			&synchronizerUpdate,
			container.GetOutputDecoder(ctx),
			synchronizerAppCreate,
			synchronizerReport,
			synchronizerOutputUpdate,
			container.GetRawOutputRefRepository(ctx),
			synchronizerOutputCreate,
			synchronizerInputCreate,
			synchronizerOutputExecuted,
		)
		w.Workers = append(w.Workers, synchronizerWorker)
	}

	cleanSync := synchronizer.NewCleanSynchronizer(container.GetSyncRepository(ctx), nil)
	w.Workers = append(w.Workers, cleanSync)

	slog.InfoContext(ctx, "Listening", "port", opts.HttpPort)
	return w
}

func NewAbiDecoder(abi *abi.ABI) {
	panic("unimplemented")
}

func CreateDBInstance(ctx context.Context, opts BootstrapOpts) *sqlx.DB {
	var db *sqlx.DB
	if opts.DbImplementation == "postgres" {
		slog.InfoContext(ctx, "Using PostGres DB ...")
		postgresHost := os.Getenv("POSTGRES_HOST")
		postgresPort := os.Getenv("POSTGRES_PORT")
		postgresDataBase := os.Getenv("POSTGRES_DB")
		postgresUser := os.Getenv("POSTGRES_USER")
		postgresPassword := os.Getenv("POSTGRES_PASSWORD")
		connectionString := fmt.Sprintf("host=%s port=%s user=%s "+
			"dbname=%s password=%s sslmode=disable",
			postgresHost, postgresPort, postgresUser,
			postgresDataBase, postgresPassword)
		dbUrl, ok := os.LookupEnv("POSTGRES_GRAPHQL_DB_URL")
		if ok {
			connectionString = dbUrl
		} else {
			slog.WarnContext(ctx, "The environment variables POSTGRES_HOST, POSTGRES_PORT, POSTGRES_DB, POSTGRES_USER, and POSTGRES_PASSWORD are deprecated. Please use POSTGRES_GRAPHQL_DB_URL instead.")
		}
		db = sqlx.MustConnect("postgres", connectionString)
		configureConnectionPool(ctx, db)
	} else {
		db = handleSQLite(ctx, opts)
	}
	return db
}

// configureConnectionPool sets the connection pool settings for the database connection.
// The following environment variables are used to configure the connection pool:
// - DB_MAX_OPEN_CONNS: Maximum number of open connections to the database
// - DB_MAX_IDLE_CONNS: Maximum number of idle connections in the pool
// - DB_CONN_MAX_LIFETIME: Maximum amount of time a connection may be reused
// - DB_CONN_MAX_IDLE_TIME: Maximum amount of time a connection may be idle
func configureConnectionPool(ctx context.Context, db *sqlx.DB) {
	defaultConnMaxLifetime := int(DefaultConnMaxLifetime.Seconds())
	defaultConnMaxIdleTime := int(DefaultConnMaxIdleTime.Seconds())

	maxOpenConns := getEnvInt(ctx, "DB_MAX_OPEN_CONNS", DefaultMaxOpenConnections)
	maxIdleConns := getEnvInt(ctx, "DB_MAX_IDLE_CONNS", DefaultMaxIdleConnections)
	connMaxLifetime := getEnvInt(ctx, "DB_CONN_MAX_LIFETIME", defaultConnMaxLifetime)
	connMaxIdleTime := getEnvInt(ctx, "DB_CONN_MAX_IDLE_TIME", defaultConnMaxIdleTime)
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Second)
	db.SetConnMaxIdleTime(time.Duration(connMaxIdleTime) * time.Second)
}

func getEnvInt(ctx context.Context, envName string, defaultValue int) int {
	value, exists := os.LookupEnv(envName)
	if !exists {
		return defaultValue
	}
	intValue, err := cast.ToIntE(value)
	if err != nil {
		slog.ErrorContext(ctx, "configuration error", "envName", envName, "value", value)
		panic(err)
	}
	return intValue
}

func handleSQLite(ctx context.Context, opts BootstrapOpts) *sqlx.DB {
	slog.InfoContext(ctx, "Using SQLite ...")
	sqliteFile := opts.SqliteFile
	if sqliteFile == "" {
		sqlitePath, err := os.MkdirTemp("", "nonodo-db-*")
		if err != nil {
			panic(err)
		}
		sqliteFile = path.Join(sqlitePath, "nonodo.sqlite3")
		slog.DebugContext(ctx, "SQLite3 file created", "path", sqliteFile)
	}

	return sqlx.MustConnect("sqlite3", sqliteFile)
}
