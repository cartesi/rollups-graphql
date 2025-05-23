// (c) Cartesi and individual authors (see AUTHORS)
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/golang-migrate/migrate"
	mig "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*
var content embed.FS

const ExpectedVersion uint = 1

type Schema struct {
	migrate *mig.Migrate
	ctx     context.Context
}

func New(ctx context.Context, postgresEndpoint string) (*Schema, error) {
	driver, err := iofs.New(content, "migrations")
	if err != nil {
		return nil, err
	}

	migrate, err := mig.NewWithSourceInstance("iofs", driver, postgresEndpoint)
	if err != nil {
		return nil, err
	}

	return &Schema{migrate: migrate, ctx: ctx}, nil
}

func NewWithPool(ctx context.Context, pool *pgxpool.Pool) (*Schema, error) {
	source, err := iofs.New(content, "migrations")
	if err != nil {
		return nil, err
	}

	db := stdlib.OpenDBFromPool(pool)
	driver, err := pgx.WithInstance(db, &pgx.Config{})
	if err != nil {
		return nil, fmt.Errorf("could not instantiate pgx migrate driver: %v", err)
	}

	migrate, err := mig.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return nil, err
	}

	return &Schema{migrate: migrate, ctx: ctx}, nil
}

func (s *Schema) Version() (uint, error) {
	version, _, err := s.migrate.Version()
	if err != nil && errors.Is(err, migrate.ErrNilVersion) {
		return version, fmt.Errorf("No valid database schema found")
	}
	return version, err
}

func (s *Schema) Upgrade() error {
	if err := s.migrate.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func (s *Schema) Downgrade() error {
	if err := s.migrate.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func (s *Schema) Close() {
	source, db := s.migrate.Close()
	if source != nil {
		slog.ErrorContext(s.ctx, "Error releasing migration sources", "error", source)
	}
	if db != nil {
		slog.ErrorContext(s.ctx, "Error closing db connection", "error", db)
	}
}

func (s *Schema) ValidateVersion() (uint, error) {
	version, err := s.Version()
	if err != nil {
		return 0, err
	}

	if version != ExpectedVersion {
		format := "Database schema version mismatch. Expected %d but it is %d"
		return 0, fmt.Errorf(format, ExpectedVersion, version)
	}
	return version, nil
}

func main() {
	var s *Schema
	var err error
	ctx := context.Background()

	postgresEndpoint := os.Getenv("CARTESI_GRAPHQL_DATABASE_CONNECTION")

	uri, err := url.Parse(postgresEndpoint)
	if err == nil {
		uri.User = nil
	} else {
		slog.ErrorContext(ctx, "Failed to parse PostgresEndpoint.", "error", err)
		os.Exit(1)
	}

	maxRetry := 5

	// nolint
	retrySleep := 5 * time.Second

	for i := 0; i < maxRetry; i++ {
		s, err = New(ctx, postgresEndpoint)
		if err == nil {
			break
		}
		slog.WarnContext(ctx, "Connection to database failed. Trying again.", "PostgresEndpoint", uri.String())
		if i == maxRetry-1 {
			slog.ErrorContext(ctx, "Failed to connect to database.", "error", err)
			os.Exit(1)
		}
		time.Sleep(retrySleep)
	}
	defer s.Close()

	err = s.Upgrade()
	if err != nil {
		slog.ErrorContext(ctx, "Error while upgrading database schema", "error", err)
		os.Exit(1)
	}

	version, err := s.ValidateVersion()
	if err != nil {
		slog.ErrorContext(ctx, "Error while validating database schema version", "error", err)
		os.Exit(1)
	}

	slog.InfoContext(ctx, "Database Schema successfully Updated.", "version", version)
}
