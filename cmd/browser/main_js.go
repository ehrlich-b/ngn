//go:build js && wasm

package main

import "syscall/js"

func main() {
	js.Global().Set("ngn", js.FuncOf(func(this js.Value, args []js.Value) any { return string(execute([]byte(args[0].String()))) }))
	select {}
}
