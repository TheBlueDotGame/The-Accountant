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
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"

	"github.com/bartossh/Computantis/fileoperations"
	"github.com/bartossh/Computantis/transaction"
	"github.com/bartossh/Computantis/wallet"
	"github.com/bartossh/Computantis/walletmiddleware"
)

var helper *walletmiddleware.Client

func new() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		serverAddress := args[0].String()
		helper = walletmiddleware.NewClient(serverAddress, 5*time.Second, wallet.Helper{}, fileoperations.Helper{}, wallet.New)
		return nil
	})
}

func validateAPI() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		if err := helper.ValidateApiVersion(); err != nil {
			return err.Error()
		}
		return "Ok"
	})
}

func newWallet() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		token := args[0].String()
		if err := helper.NewWallet(token); err != nil {
			return err.Error()
		}
		return "Ok"
	})
}

func getAddress() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		addr, err := helper.Address()
		if err != nil {
			fmt.Println(err.Error())
			return ""
		}
		return addr
	})
}

func proposeTransaction() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		receiverAddr := args[0].String()
		subject := args[1].String()
		data := args[2].String()
		if err := helper.ProposeTransaction(receiverAddr, subject, []byte(data)); err != nil {
			return err.Error()
		}
		return "Ok"
	})
}

func confirmTransaction() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		jsonTrx := args[0].String()
		var trx transaction.Transaction
		if err := json.Unmarshal([]byte(jsonTrx), &trx); err != nil {
			return err.Error()
		}
		if err := helper.ConfirmTransaction(&trx); err != nil {
			return err.Error()
		}
		return "Ok"
	})
}

func readWaitingTransactions() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		trxs, err := helper.ReadWaitingTransactions()
		if err != nil {
			fmt.Println(err.Error())
			return []string{}
		}
		result := make([]string, len(trxs))
		for _, trx := range trxs {
			jsonTrx, err := json.Marshal(trx)
			if err != nil {
				fmt.Println(err.Error())
				return []string{}
			}
			result = append(result, string(jsonTrx))
		}

		return result
	})
}

func readIssuedTransactions() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		trxs, err := helper.ReadIssuedTransactions()
		if err != nil {
			fmt.Println(err.Error())
			return []string{}
		}
		result := make([]string, len(trxs))
		for _, trx := range trxs {
			jsonTrx, err := json.Marshal(trx)
			if err != nil {
				fmt.Println(err.Error())
				return []string{}
			}
			result = append(result, string(jsonTrx))
		}

		return result
	})
}

func saveWallet() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		if err := helper.SaveWalletToFile(); err != nil {
			return err.Error()
		}
		return "Ok"
	})
}

func readWallet() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		if err := helper.ReadWalletFromFile(); err != nil {
			return err.Error()
		}
		return "Ok"
	})
}

func flush() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		helper.FlushWalletFromMemory()
		return "Ok"
	})
}

func main() {
	fmt.Println("WebAssembly wallet starting ...")
	js.Global().Set("new", new())
	js.Global().Set("validateAPI", validateAPI)
	js.Global().Set("newWallet", newWallet)
	js.Global().Set("getAddress", getAddress)
	js.Global().Set("proposeTransaction", proposeTransaction)
	js.Global().Set("confirmTransaction", confirmTransaction)
	js.Global().Set("readWaitingTransactions", readWaitingTransactions)
	js.Global().Set("readIssuedTransactions", readIssuedTransactions)
	js.Global().Set("saveWallet", saveWallet)
	js.Global().Set("readWallet", readWallet)
	js.Global().Set("flush", flush)
	<-make(chan bool)
	fmt.Println("WebAssembly wallet closing ...")
}
