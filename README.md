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
export CARTESI_GRAPHQL_DATABASE_CONNECTION="postgres://postgres:password@127.0.0.1:5432/hlgraphql?sslmode=disable"
export CARTESI_DATABASE_CONNECTION="postgres://postgres:password@localhost:5432/rollupsdb?sslmode=disable"
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
export CARTESI_GRAPHQL_DATABASE_CONNECTION="postgres://postgres:password@localhost:5432/hlgraphql?sslmode=disable"
export CARTESI_DATABASE_CONNECTION="postgres://postgres:password@localhost:5432/rollupsdb?sslmode=disable"
./cartesi-rollups-graphql
```

## Environment Variables

The following environment variables are used for PostgreSQL configuration:

- `CARTESI_GRAPHQL_DATABASE_CONNECTION`: URL for the PostgreSQL database used by GraphQL.
- `CARTESI_DATABASE_CONNECTION`: URL for the PostgreSQL database used by the node.
- `DB_MAX_OPEN_CONNS`: Maximum number of open connections to the database (default: 25).
- `DB_MAX_IDLE_CONNS`: Maximum number of idle connections in the pool (default: 10).
- `DB_CONN_MAX_LIFETIME`: Maximum amount of time a connection may be reused (default: 1800 seconds).
- `DB_CONN_MAX_IDLE_TIME`: Maximum amount of time a connection may be idle (default: 300 seconds).

## Contributors

[![Contributors](https://contributors-img.firebaseapp.com/image?repo=cartesi/rollups-graphql)](https://github.com/cartesi/rollups-graphql/graphs/contributors)

Made with [contributors-img](https://contributors-img.firebaseapp.com).

## Release

New releases are created using the [Changesets](https://github.com/changesets/changesets/blob/main/packages/cli/README.md) library, which is already set up in this project.

### How to do a standard release

#### Manual steps

1. **Workflow permissions**:

   - Ensure this repository has "Read and write permissions" enabled under the "Workflow permissions" section in the repository settings.

2. **Create a changeset when you're ready to release a new version**:

   ```bash
   npx changeset
   ```

   - Select the type of change (patch, minor, major)
   - Write a short description of the change
   - Make sure a `.md` file is automatically created inside the `.changeset/` directory

3. **'push' your changes to remote repo**

   No github action is trigered until here.

4. **New Pull Request**

   Create a pull request from your branch into `main` branch.

#### Automatic procedures

After merges/commits into `main` branch in a changeset state (with an `md` file in `.changeset` directory):

1. `Release Pull Request` workflow job is automatically triggered
2. Changesets will create a release pull request. In this PR some files will be updated by changeset bot to bump the version number
3. Once you merge this PR, `Release Pull Request` workflow job will run again, and this time a new git tag will be created and pushed:
   - `package.json` version will be updated
   - A GitHub tag will be created
   - CHANGELOG.md will be updated

### How to pre release

See the [docs](https://github.com/changesets/changesets/blob/main/docs/prereleases.md).
