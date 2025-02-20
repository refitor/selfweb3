# !/bin/bash
go mod init selfweb3
go mod tidy

# build selfweb3.wasm
cd ./backend/wasm
rm ../../vweb/public/selfweb3.wasm
GOOS=js GOARCH=wasm go build -ldflags="-w -s" -o ../../rsweb/selfweb3.wasm
cd ../../

# # build web
cd ./vweb
rm -rf ../rsweb/app
yarn install
yarn run build
cd ../

# build selfweb3
# go build -o selfweb3 main.go
