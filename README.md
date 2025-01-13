# Cartesi's GraphQL

![CI](https://github.com/cartesi/rollups-graphql/actions/workflows/ci.yaml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/cartesi/rollups-graphql)](https://goreportcard.com/report/github.com/cartesi/rollups-graphql)

[Technical Vision Forum Discussion](https://governance.cartesi.io/t/convenience-layer-for-voucher-management-on-cartesi/401)

[Internal docs](./docs/convenience.md)

## Description

Exposes the GraphQL reader API in the endpoint `http://127.0.0.1:8080/graphql`.
You may access this address to use the GraphQL interactive playground in your web browser.
You can also make POST requests directly to the GraphQL API.
For instance, the command below gets the number of inputs.

```sh
QUERY='query { inputs { totalCount } }'; \
curl \
    -X POST \
    -H 'Content-Type: application/json' \
    -d "{\"query\": \"$QUERY\"}" \
    http://127.0.0.1:8080/graphql
```

## Connecting to Postgres locally

Start a Postgres instance locally using docker compose.

```sh
make up-db-raw
```

```sh
export POSTGRES_GRAPHQL_DB_URL="postgres://postgres:password@localhost:5432/rlgraphql?sslmode=disable"
export POSTGRES_NODE_DB_URL="postgres://postgres:password@localhost:5432/rollupsdb?sslmode=disable"
./cartesi-rollups-graphql
```

## Environment Variables

The following environment variables are used for PostgreSQL configuration:

- `POSTGRES_GRAPHQL_DB_URL`: URL for the PostgreSQL database used by GraphQL.
- `POSTGRES_NODE_DB_URL`: URL for the PostgreSQL database used by the node.
- `DB_MAX_OPEN_CONNS`: Maximum number of open connections to the database (default: 25).
- `DB_MAX_IDLE_CONNS`: Maximum number of idle connections in the pool (default: 10).
- `DB_CONN_MAX_LIFETIME`: Maximum amount of time a connection may be reused (default: 1800 seconds).
- `DB_CONN_MAX_IDLE_TIME`: Maximum amount of time a connection may be idle (default: 300 seconds).

## Contributors

[![Contributors](https://contributors-img.firebaseapp.com/image?repo=cartesi/rollups-graphql)](https://github.com/cartesi/rollups-graphql/graphs/contributors)

Made with [contributors-img](https://contributors-img.firebaseapp.com).
