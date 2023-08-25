# !/bin/bash

# build selfweb3.wasm
cd ./backend/wasm
# go mod init selfweb3
# go mod tidy
rm ../../web3/public/selfweb3.wasm
GOOS=js GOARCH=wasm go build -ldflags="-w -s" -o ../../web3/public/selfweb3.wasm
cd ../../

# # build web
cd ./web3
rm -rf ../rsweb/app
yarn install
yarn run build
cd ../

# build selfweb3
go build -o selfweb3 main.go
