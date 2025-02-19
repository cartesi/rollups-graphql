#!/bin/zsh
set -e

rm -rf ./rollups-node

git clone -b feature/new-build --recurse-submodules https://github.com/cartesi/rollups-node.git

if [ ! -d "rollups-espresso-reader" ]; then
git clone -b main --recurse-submodules https://github.com/cartesi/rollups-espresso-reader.git
fi

docker stop $(docker ps -q) || true

docker buildx prune --all --force && docker system prune --volumes --force

docker run -d --rm --name postgres -p 5432:5432 -e POSTGRES_PASSWORD=password -e POSTGRES_DB=rollupsdb postgres:16-alpine

IS_WSL=false
if [[ $(grep -i Microsoft /proc/version) ]]; then
WSL=true;
fi

echo "Migrate DB node v2"
cd rollups-node
if [ "$WSL" = true ]; then
    export PATH=$(printf '%q' $(printenv PATH))
fi
eval $(make env)
export CGO_CFLAGS="-D_GNU_SOURCE -D__USE_MISC"
go run dev/migrate/main.go
cd -

echo "Migrate DB Espresso"
cd rollups-espresso-reader
eval $(make env)
make migrate
make generate-db
cd -

echo "Build image"
#docker build -t espresso .
docker build -t espresso -f ./ci/Dockerfile .

echo "Run Anvil"
cd rollups-node
make devnet
make run-devnet
cd -

# export $(grep -v '^#' env.nodev2-sepolia | xargs)
export CARTESI_BLOCKCHAIN_HTTP_ENDPOINT=https://eth-sepolia.g.alchemy.com/v2/9hjbdwjACHkHf1j01yE2j7Q9G9J1VsC9
export CARTESI_BLOCKCHAIN_WS_ENDPOINT=wss://eth-sepolia.g.alchemy.com/v2/9hjbdwjACHkHf1j01yE2j7Q9G9J1VsC9

# go test --timeout 1m -p 1 ./...
export ESPRESSO_STARTING_BLOCK=$(curl -s https://query.decaf.testnet.espresso.network/v0/status/block-height)

docker run --env-file ./ci/env.nodev2-local --rm --network=host --name c_espresso espresso
# docker run --env-file ./ci/env.nodev2-sepolia \
#   -e CARTESI_BLOCKCHAIN_HTTP_ENDPOINT=$CARTESI_BLOCKCHAIN_HTTP_ENDPOINT \
#   -e CARTESI_BLOCKCHAIN_WS_ENDPOINT=$CARTESI_BLOCKCHAIN_WS_ENDPOINT \
#   --rm --network=host --name c_espresso espresso

exit 0
