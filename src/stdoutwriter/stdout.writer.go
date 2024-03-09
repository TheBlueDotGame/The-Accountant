package stdoutwriter

import "fmt"

type Logger struct{}

func (l Logger) Write(p []byte) (n int, err error) {
	fmt.Println(string(p))
	return len(p), nil
}
