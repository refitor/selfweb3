# !/bin/bash

# # build selfweb3.wasm
# cd ./backend
# go mod init selfweb3
# go mod tidy
# rm ../web/public/selfweb3.wasm
# GOOS=js GOARCH=wasm go build -ldflags="-w -s" -o ../web/public/selfweb3.wasm
# cd ../

# build web
cd ./web
yarn install
yarn run build
rm -rf ../rsweb/app
mv ../app ../rsweb/