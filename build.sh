# !/bin/bash

# build selfweb3.wasm
cd ./backend
go mod init selfweb3
go mod tidy
rm ../web3/public/selfweb3.wasm
GOOS=js GOARCH=wasm go build -ldflags="-w -s" -o ../web3/public/selfweb3.wasm
cd ../

# build web
cd ./web3
yarn install
yarn run build