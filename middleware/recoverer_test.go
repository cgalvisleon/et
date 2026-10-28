package middleware

import (
	"testing"
)

/**
* TestPrettyStack: Una pila real de Go (con "panic({0x...})" y la línea "goroutine N
* [running]:") se decora sin hacer panic.
**/
func TestPrettyStack(t *testing.T) {
	stack := []byte(`goroutine 7 [running]:
runtime/debug.Stack()
	/usr/local/go/src/runtime/debug/stack.go:26 +0x5e
panic({0x1046a2e40?, 0x140001b2000?})
	/usr/local/go/src/runtime/panic.go:792 +0x124
github.com/josefina/internal/jdb.(*Model).insert(0x140001b4000)
	/src/josefina/internal/jdb/model.go:120 +0x40
main.main()
	/src/josefina/main.go:10 +0x20
`)
	for _, rvr := range []any{"boom", nil} {
		if _, err := (prettyStack{}).parse(stack, rvr); err != nil {
			t.Fatalf("parse: %v", err)
		}
	}
	for _, line := range []string{"panic({0x1, 0x2})", "main()", "a/b/c(0x1)"} {
		if _, err := (prettyStack{}).decorateFuncCallLine(line, 0); err != nil {
			t.Fatalf("%s: %v", line, err)
		}
	}
}
