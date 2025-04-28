// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This package is responsible for serving the GraphQL reader API.
package reader

//go:generate go run github.com/99designs/gqlgen generate

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	cModel "github.com/cartesi/rollups-graphql/v2/pkg/convenience/model"
	"github.com/cartesi/rollups-graphql/v2/pkg/convenience/services"
	"github.com/cartesi/rollups-graphql/v2/pkg/reader/graph"
	"github.com/cartesi/rollups-graphql/v2/pkg/reader/loaders"
	"github.com/labstack/echo/v4"
)

// Register the GraphQL reader API to echo.
func Register(
	ctx context.Context,
	e *echo.Echo,
	convenienceService *services.ConvenienceService,
	adapter Adapter,
) {
	resolver := Resolver{
		convenienceService,
		adapter,
	}
	config := graph.Config{Resolvers: &resolver}
	schema := graph.NewExecutableSchema(config)
	graphqlHandler := handler.NewDefaultServer(schema)
	playgroundHandler := playground.Handler("GraphQL", "/graphql")
	e.POST("/graphql", func(c echo.Context) error {
		graphqlHandler.ServeHTTP(c.Response(), c.Request())
		return nil
	})
	e.POST("/graphql/:appContract", func(c echo.Context) error {
		appContract := c.Param("appContract")
		slog.DebugContext(ctx, "path parameter received: ", "app_contract", appContract)
		ctx := context.WithValue(c.Request().Context(), cModel.AppContractKey, appContract)
		loader := loaders.NewLoaders(
			convenienceService.ReportRepository,
			convenienceService.VoucherRepository,
			convenienceService.NoticeRepository,
			convenienceService.InputRepository,
		)
		ctx = context.WithValue(ctx, loaders.LoadersKey, loader)
		c.SetRequest(c.Request().WithContext(ctx))
		graphqlHandler.ServeHTTP(c.Response(), c.Request())
		return nil
	})
	e.GET("/graphql", func(c echo.Context) error {
		playgroundHandler.ServeHTTP(c.Response(), c.Request())
		return nil
	})
	e.GET("/graphql/:appContract", func(c echo.Context) error {
		appContract := c.Param("appContract")
		slog.DebugContext(ctx, "graphql playground", "appContract", appContract)
		playgroundHandler := playground.Handler("GraphQL",
			fmt.Sprintf("/graphql/%s", appContract),
		)
		playgroundHandler.ServeHTTP(c.Response(), c.Request())
		return nil
	})
}
