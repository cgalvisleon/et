package middleware

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/cgalvisleon/et/request"
	lg "github.com/cgalvisleon/et/stdrout"
)

/**
* logBufPool reuses bytes.Buffer instances across requests to reduce GC pressure.
**/
var logBufPool = sync.Pool{
	New: func() interface{} { return &bytes.Buffer{} },
}

var (
	/**
	* LogEntryCtxKey is the context key used to store the LogEntry for a request.
	**/
	LogEntryCtxKey = request.ContextKey("LogEntry")

	/**
	* DefaultLogger is the package-level Logger middleware. Replace to customize logging.
	**/
	DefaultLogger func(next http.Handler) http.Handler
)

/**
* Logger middleware logs the start and end of each request with method, path,
* status, size, and elapsed time. Outputs color when stdout is a TTY.
* Must be registered before Recoverer in the middleware chain.
* @param next http.Handler
* @return http.Handler
**/
func Logger(next http.Handler) http.Handler {
	return DefaultLogger(next)
}

/**
* RequestLogger returns a middleware that logs requests using the given LogFormatter.
* @param f LogFormatter
* @return func(next http.Handler) http.Handler
**/
func RequestLogger(f LogFormatter) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			metric := NewMetric(r)
			metric.CallSearchTime()
			w.Header().Set("ServiceId", metric.ServiceId)
			ww := &ResponseWriterWrapper{ResponseWriter: w, StatusCode: http.StatusOK}
			entry := f.NewLogEntry(r)
			wr := WithLogEntry(r, entry)

			next.ServeHTTP(ww, wr)
			metric.DoneHTTP(ww)
		}

		return http.HandlerFunc(fn)
	}
}

/**
* LogFormatter creates a new LogEntry at the start of each request.
**/
type LogFormatter interface {
	NewLogEntry(r *http.Request) LogEntry
}

/**
* LogEntry writes the final log line when a request completes.
**/
type LogEntry interface {
	Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{})
	Panic(v interface{}, stack []byte)
}

/**
* GetLogEntry retrieves the LogEntry stored in the request context.
* @param r *http.Request
* @return LogEntry
**/
func GetLogEntry(r *http.Request) LogEntry {
	entry, _ := r.Context().Value(LogEntryCtxKey).(LogEntry)
	return entry
}

/**
* WithLogEntry stores a LogEntry in the request context.
* @param r *http.Request, entry LogEntry
* @return *http.Request
**/
func WithLogEntry(r *http.Request, entry LogEntry) *http.Request {
	r = r.WithContext(context.WithValue(r.Context(), LogEntryCtxKey, entry))
	return r
}

/**
* LoggerInterface is satisfied by stdlib log.Logger and compatible loggers.
**/
type LoggerInterface interface {
	Print(v ...interface{})
}

/**
* DefaultLogFormatter is the default LogFormatter implementation.
**/
type DefaultLogFormatter struct {
	Logger  LoggerInterface
	NoColor bool
}

/**
* NewLogEntry creates a LogEntry for the request, reusing a pooled buffer.
* @param r *http.Request
* @return LogEntry
**/
func (l *DefaultLogFormatter) NewLogEntry(r *http.Request) LogEntry {
	buf := logBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	entry := &defaultLogEntry{
		DefaultLogFormatter: l,
		request:             r,
		buf:                 buf,
	}

	var w *string
	reqID := GetReqID(r.Context())
	if reqID != "" {
		lg.Color(w, lg.Yellow, "[%s] ", reqID)
	}
	lg.Color(w, lg.Cyan, "")
	lg.Color(w, lg.Purple, "[%s]: ", r.Method)

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	lg.Color(w, lg.Cyan, "%s://%s%s %s ", scheme, r.Host, r.RequestURI, r.Proto)

	entry.buf.WriteString("from ")
	entry.buf.WriteString(r.RemoteAddr)
	entry.buf.WriteString(" - ")

	return entry
}

/**
* defaultLogEntry holds per-request log state. buf is returned to logBufPool after Write.
**/
type defaultLogEntry struct {
	*DefaultLogFormatter
	request *http.Request
	buf     *bytes.Buffer
}

/**
* Write logs the completed request: status, size, and elapsed time.
* Returns buf to the pool after printing.
* @param status int, bytes int, header http.Header, elapsed time.Duration, extra interface{}
**/
func (l *defaultLogEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	var w *string
	switch {
	case status < 200:
		lg.Color(w, lg.Green, "%03d", status)
	case status < 300:
		lg.Color(w, lg.Green, "%03d", status)
	case status < 400:
		lg.Color(w, lg.Cyan, "%03d", status)
	case status < 500:
		lg.Color(w, lg.Yellow, "%03d", status)
	default:
		lg.Color(w, lg.Red, "%03d", status)
	}

	lg.Color(w, lg.Blue, " %dB", bytes)

	l.buf.WriteString(" in ")
	if elapsed < 500*time.Millisecond {
		lg.Color(w, lg.Green, "%s", elapsed)
	} else if elapsed < 5*time.Second {
		lg.Color(w, lg.Yellow, "%s", elapsed)
	} else {
		lg.Color(w, lg.Red, "%s", elapsed)
	}

	l.Logger.Print(l.buf.String())
	logBufPool.Put(l.buf)
	l.buf = nil
}

/**
* Panic prints a formatted stack trace for a recovered panic.
* @param v interface{}, stack []byte
**/
func (l *defaultLogEntry) Panic(v interface{}, stack []byte) {
	PrintPrettyStack(v)
}

func init() {
	color := true
	if runtime.GOOS == "windows" {
		color = false
	}
	DefaultLogger = RequestLogger(&DefaultLogFormatter{Logger: log.New(os.Stdout, "", log.LstdFlags), NoColor: !color})
}
