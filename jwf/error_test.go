package jwf

import (
	"errors"
	"testing"
)

/**
* TestScriptError: a script error comes back as its message, without the "at <eval>:line:col" of goja
**/
func TestScriptError(t *testing.T) {
	bindings := map[string]any{
		"fail": func() error { return errors.New("from go") },
	}
	for _, tc := range []struct {
		code string
		want string
	}{
		{`throw new Error("La contraseña no es correcta")`, "La contraseña no es correcta"},
		{`throw "plain text"`, "plain text"},
		{`fail()`, "from go"},
		{`undefinedName + 1`, "ReferenceError: undefinedName is not defined"},
		{`null.x`, "TypeError: Cannot read property 'x' of undefined"},
	} {
		_, _, err := Run(tc.code, bindings)
		if err == nil {
			t.Errorf("%s: no error", tc.code)
			continue
		}
		if err.Error() != tc.want {
			t.Errorf("%s: %q, want %q", tc.code, err.Error(), tc.want)
		}
	}
}
