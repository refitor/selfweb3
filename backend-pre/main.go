package main

import (
	"selfweb3/server"
	"selfweb3/wasm"
)

var BuildType = "server"

func main() {
	if BuildType == "server" {
		server.Run()
	} else {
		wasm.Run()
	}
}
