package middleware

// The original work was derived from Goji's middleware, source:
// https://github.com/zenazn/goji/tree/master/web/middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"github.com/cgalvisleon/et/mistake"
	lg "github.com/cgalvisleon/et/stdrout"
)

// Recoverer is a middleware that recovers from panics, logs the panic (and a
// backtrace), and returns a HTTP 500 (Internal Server Error) status if
// possible. Recoverer prints a request ID if one is provided.
//
// Alternatively, look at https://github.com/pressly/lg middleware pkgs.
func Recoverer(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil && rvr != http.ErrAbortHandler {

				logEntry := GetLogEntry(r)
				if logEntry != nil {
					logEntry.Panic(rvr, debug.Stack())
				} else {
					PrintPrettyStack(rvr)
				}

				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func PrintPrettyStack(rvr interface{}) {
	debugStack := debug.Stack()
	s := prettyStack{}
	out, err := s.parse(debugStack, rvr)
	if err == nil {
		os.Stderr.Write(out)
	} else {
		os.Stderr.Write(debugStack)
	}
}

type prettyStack struct {
}

func (s prettyStack) parse(debugStack []byte, rvr interface{}) ([]byte, error) {
	var err error
	buf := &bytes.Buffer{}

	lg.CW(buf, lg.BRed, "\n")
	lg.CW(buf, lg.BCyan, " panic: ")
	lg.CW(buf, lg.BBlue, "%v", rvr)
	lg.CW(buf, lg.BWhite, "\n \n")

	// process debug stack info
	stack := strings.Split(string(debugStack), "\n")
	lines := []string{}

	// locate panic line, as we may have nested panics
	for i := len(stack) - 1; i > 0; i-- {
		lines = append(lines, stack[i])
		if strings.HasPrefix(stack[i], "panic(0x") {
			lines = lines[0 : len(lines)-2] // remove boilerplate
			break
		}
	}

	// reverse
	for i := len(lines)/2 - 1; i >= 0; i-- {
		opp := len(lines) - 1 - i
		lines[i], lines[opp] = lines[opp], lines[i]
	}

	// decorate
	for i, line := range lines {
		lines[i], err = s.decorateLine(line, i)
		if err != nil {
			return nil, err
		}
	}

	for _, l := range lines {
		fmt.Fprintf(buf, "%s", l)
	}
	return buf.Bytes(), nil
}

func (s prettyStack) decorateLine(line string, num int) (string, error) {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "\t") || strings.Contains(line, ".go:") {
		return s.decorateSourceLine(line, num)
	} else if strings.HasSuffix(line, ")") {
		return s.decorateFuncCallLine(line, num)
	} else {
		if strings.HasPrefix(line, "\t") {
			return strings.Replace(line, "\t", "      ", 1), nil
		} else {
			return fmt.Sprintf("    %s\n", line), nil
		}
	}
}

func (s prettyStack) decorateFuncCallLine(line string, num int) (string, error) {
	idx := strings.LastIndex(line, "(")
	if idx < 0 {
		return "", mistake.New("not a func call line")
	}

	buf := &bytes.Buffer{}
	pkg := line[0:idx]
	addr := line[idx:]
	method := ""

	idx = strings.LastIndex(pkg, string(os.PathSeparator))
	if idx < 0 {
		idx = strings.Index(pkg, ".")
		method = pkg[idx:]
		pkg = pkg[0:idx]
	} else {
		method = pkg[idx+1:]
		pkg = pkg[0 : idx+1]
		idx = strings.Index(method, ".")
		pkg += method[0:idx]
		method = method[idx:]
	}
	pkgColor := lg.NYellow
	methodColor := lg.BGreen

	if num == 0 {
		lg.CW(buf, lg.BRed, " -> ")
		pkgColor = lg.BMagenta
		methodColor = lg.BRed
	} else {
		lg.CW(buf, lg.BWhite, "    ")
	}
	lg.CW(buf, pkgColor, "%s", pkg)
	lg.CW(buf, methodColor, "%s\n", method)
	lg.CW(buf, lg.BBlack, "%s", addr)

	return buf.String(), nil
}

func (s prettyStack) decorateSourceLine(line string, num int) (string, error) {
	idx := strings.LastIndex(line, ".go:")
	if idx < 0 {
		return "", mistake.New("not a source line")
	}

	buf := &bytes.Buffer{}
	path := line[0 : idx+3]
	lineno := line[idx+3:]

	idx = strings.LastIndex(path, string(os.PathSeparator))
	dir := path[0 : idx+1]
	file := path[idx+1:]

	idx = strings.Index(lineno, " ")
	if idx > 0 {
		lineno = lineno[0:idx]
	}
	fileColor := lg.BCyan
	lineColor := lg.BGreen

	if num == 1 {
		lg.CW(buf, lg.BRed, " ->   ")
		fileColor = lg.BRed
		lineColor = lg.BMagenta
	} else {
		lg.CW(buf, lg.BWhite, "      ")
	}
	lg.CW(buf, lg.BWhite, "%s", dir)
	lg.CW(buf, fileColor, "%s", file)
	lg.CW(buf, lineColor, "%s", lineno)
	if num == 1 {
		lg.CW(buf, lg.BWhite, "\n")
	}
	lg.CW(buf, lg.BWhite, "\n")

	return buf.String(), nil
}
