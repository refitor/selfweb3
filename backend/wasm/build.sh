# build selfweb3.wasm
rm ../../web3/public/selfweb3.wasm
GOOS=js GOARCH=wasm go build -ldflags="-w -s" -o ../../web3/public/selfweb3.wasm
