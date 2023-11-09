package main

import (
	"encoding/json"
	"syscall/js"

	"selfweb3/backend/src/wasm"
)

func main() {
	// init
	wasm.Init()

	js.Global().Set("WasmInit", js.FuncOf(wrapWasmFunc(wasm.WasmInit)))
	js.Global().Set("WasmStatus", js.FuncOf(wrapWasmFunc(wasm.WasmStatus)))
	js.Global().Set("WasmHandle", js.FuncOf(wrapWasmFunc(wasm.WasmHandle)))
	js.Global().Set("WasmVerify", js.FuncOf(wrapWasmFunc(wasm.WasmVerify)))
	js.Global().Set("WasmPublic", js.FuncOf(wrapWasmFunc(wasm.WasmPublic)))
	js.Global().Set("WasmRegister", js.FuncOf(wrapWasmFunc(wasm.WasmRegister)))
	js.Global().Set("WasmAuthorizeCode", js.FuncOf(wrapWasmFunc(wasm.WasmAuthorizeCode)))

	select {}
}

func wrapWasmFunc(f func(datas ...string) *wasm.Response) func(this js.Value, args []js.Value) any {
	return func(this js.Value, args []js.Value) any {
		callback := args[len(args)-1]
		go func() {
			requestParams := make([]string, 0)
			for i := 0; i < len(args)-1; i++ {
				requestParams = append(requestParams, args[i].String())
			}
			responseBuf, _ := json.Marshal(f(requestParams...))
			wasm.LogDebugf("wasmFunc response: %v", string(responseBuf))
			callback.Invoke(string(responseBuf))
		}()
		return nil
	}
}
