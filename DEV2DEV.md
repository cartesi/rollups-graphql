# Developer Notes

```shell
watchexec --exts go --watch . 'make test && make lint'
```

To run just one test:

```shell
watchexec --exts go --watch . 'go test -p 1 ./... -testify.m ^TestNoDuplicateInputs$'
```

uint64 type is based on [rollups_outputs.rs](https://github.com/cartesi/rollups-node/blob/392c75972037352ecf94fb482619781b1b09083f/offchain/rollups-events/src/rollups_outputs.rs#L41)

```go
Voucher
InputIndex  uint64
OutputIndex uint64
```

Input encoded by rollups-contract V2

```text
0xcc7dee1f000000000000000000000000000000000000000000000000cc0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000007fa9385be102ac3eac297483dd6233d62b3e149600000000000000000000000000000000000000000000000000000000000000e10000000000000000000000000000000000000000000000000000000061d0c1b100000000000000000000000000000000000000000000000000000000000000e100000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000020157f9f93806730d47e3a6e8b41b4fa9d98f89ec6f53627f3abf1d171769345eb
```

## How to run the GraphQL

Run the database

```shell
docker compose -f postgres/raw/compose.yml up --wait
```

Stop with clean:

```shell
docker compose -f postgres/raw/compose.yml down --volumes --remove-orphans --rmi local
```

[http://localhost:8080/graphql](http://localhost:8080/graphql)

### Build

```shell
go build
```

Run the GraphQL flag enabled

```shell
go run . --enable-debug
```

```shell
export POSTGRES_GRAPHQL_DB_URL="postgres://postgres:password@127.0.0.1:5432/hlgraphql?sslmode=disable"
export POSTGRES_NODE_DB_URL="postgres://postgres:password@localhost:5432/rollupsdb?sslmode=disable"
go run . --http-address=0.0.0.0 --http-port 8081 --enable-debug --db-implementation postgres
```

## Environment Variables

Clear database raw:

```bash
make clean-db-raw
```

## Database Configuration

The following environment variables are used for PostgreSQL configuration:

- `POSTGRES_GRAPHQL_DB_URL`: URL for the PostgreSQL database used by GraphQL.
- `POSTGRES_NODE_DB_URL`: URL for the PostgreSQL database used by the node.
- `DB_MAX_OPEN_CONNS`: Maximum number of open connections to the database (default: 25).
- `DB_MAX_IDLE_CONNS`: Maximum number of idle connections in the pool (default: 10).
- `DB_CONN_MAX_LIFETIME`: Maximum amount of time a connection may be reused (default: 1800 seconds).
- `DB_CONN_MAX_IDLE_TIME`: Maximum amount of time a connection may be idle (default: 300 seconds).
