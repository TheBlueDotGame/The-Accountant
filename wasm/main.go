//go:build js && wasm

// build with: `GOOS=js GOARCH=wasm go build -o ./wasm/bin/wallet.wasm`
// Get wasm_exec.js: `cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" ./wasm/js/`

// your front-end html should look like this:
// ```html
// <html>
// <head>
//
//	<meta charset="utf-8"/>
//	<script src="wasm_exec.js"></script>
//	<script>
//		const go = new Go();
//		WebAssembly.instantiateStreaming(fetch("wallet.wasm"), go.importObject).then((result) => {
//			go.run(result.instance);
//		});
//	</script>
//
// </head>
// <body></body>
// </html>
// ```
package main

import (
	"fmt"
	"syscall/js"

	"github.com/bartossh/Computantis/client"
)

var c *client.Client

func createClient() js.Func {
	validate := js.FuncOf(func(this js.Value, args []js.Value) any {
		c = client.NewClient()
		return nil
	})
	return validate
}

func main() {
	fmt.Println("WebAssembly wallet starting ...")
	js.Global().Set("newClient", createClient())
	<-make(chan bool)
	fmt.Println("WebAssembly wallet closing ...")
}
