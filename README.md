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

```shell
export POSTGRES_GRAPHQL_DB_URL="postgres://postgres:password@127.0.0.1:5432/hlgraphql?sslmode=disable"
export POSTGRES_NODE_DB_URL="postgres://postgres:password@localhost:5432/rollupsdb?sslmode=disable"
go run . --http-address=0.0.0.0
```

## Running with Node V2

Build Node V2 and then start it.

Create the rollups graphql database:

```shell
docker exec -i postgres psql -U postgres -d hlgraphql < ./postgres/raw/hlgraphql.sql
```

Compile:

```shell
go build -o cartesi-rollups-graphql
```

Run the rollups graphql:

```shell
export POSTGRES_GRAPHQL_DB_URL="postgres://postgres:password@localhost:5432/hlgraphql?sslmode=disable"
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


# Release

## How to

New releases are created using [changeset](https://github.com/changesets/changesets/blob/main/packages/cli/README.md) library. This library is currently set up in this project.
In order to create new releases we need:

1- Commit all current changes in your current branch

2- Run `pnpm changeset`

3- Enter all prompted information (release type, release summary)

4- Check if an MD file is created, this file will trigger changeset github action to create new pull request asking for pump the project tag and aplly a release summary.

5- Commit all new changes

6- Git push 

7- If `release-pullrequest` Github Action Job is successfully executed a pull request will be automatically created asking merge from a _changeset auto created branch_ into your _current branch_.

8- After merging this intermediary pull request create a new one from _your current branch_ into _main_ branch. Merges into main branch will trigger the `Release` github action job and performs new tag and artifacts creation.