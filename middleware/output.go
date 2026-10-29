package middleware

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/cgalvisleon/et/envar"
	lg "github.com/cgalvisleon/et/stdrout"
)

/**
* Formats of the telemetry output (TELEMETRY_FORMAT): text is the colored line for a
* terminal, json one JSON object per request for log collectors (Loki, Grafana), off nothing.
**/
const (
	FormatText = "text"
	FormatJson = "json"
	FormatOff  = "off"
)

/**
* Which requests are written (TELEMETRY_LOG): all, or only errors —a status >= 500 or a
* latency >= LATENCY_LIMIT ms—, for nodes where a line per request costs too much.
**/
const (
	LogAll    = "all"
	LogErrors = "errors"
)

/**
* output: Where, how and which request lines are written.
**/
type output struct {
	format       string
	errorsOnly   bool
	latencyLimit float64
	w            io.Writer
	mu           sync.Mutex
	json         *slog.Logger
}

var current atomic.Pointer[output]

/**
* SetTelemetryOutput: Sets where the request lines go, in which format and which ones.
* @param w io.Writer, format string (text, json, off; empty = TELEMETRY_FORMAT), filter string (all, errors; empty = TELEMETRY_LOG)
**/
func SetTelemetryOutput(w io.Writer, format, filter string) {
	current.Store(newOutput(w, format, filter))
}

/**
* telemetryOutput: The current output; the first call builds it on stdout from
* TELEMETRY_FORMAT (read then, so a .env loaded at start counts).
* @return *output
**/
func telemetryOutput() *output {
	if o := current.Load(); o != nil {
		return o
	}
	current.CompareAndSwap(nil, newOutput(os.Stdout, "", ""))
	return current.Load()
}

/**
* newOutput: Builds an output; an empty or unknown format or filter takes its variable.
* @param w io.Writer, format string, filter string
* @return *output
**/
func newOutput(w io.Writer, format, filter string) *output {
	format = strings.ToLower(format)
	if !validFormat(format) {
		format = defaultFormat()
	}
	filter = strings.ToLower(filter)
	if filter != LogAll && filter != LogErrors {
		filter = strings.ToLower(envar.GetStr("TELEMETRY_LOG", LogAll))
	}
	return &output{
		format:       format,
		errorsOnly:   filter == LogErrors,
		latencyLimit: float64(envar.GetInt("LATENCY_LIMIT", 1000)),
		w:            w,
		json:         slog.New(slog.NewJSONHandler(w, nil)),
	}
}

/**
* accepts: Whether the request goes to the output (format not off and, with errors only,
* a failed or slow request).
* @param s *Metrics
* @return bool
**/
func (o *output) accepts(s *Metrics) bool {
	if o.format == FormatOff {
		return false
	}
	return !o.errorsOnly || s.StatusCode >= http.StatusInternalServerError || s.Latency >= o.latencyLimit
}

/**
* defaultFormat: TELEMETRY_FORMAT if it is valid; otherwise text on a terminal and json
* elsewhere (a container, where a collector reads stdout).
* @return string
**/
func defaultFormat() string {
	format := strings.ToLower(envar.GetStr("TELEMETRY_FORMAT", ""))
	if validFormat(format) {
		return format
	}
	if lg.IsTTY {
		return FormatText
	}
	return FormatJson
}

/**
* validFormat: Whether format is text, json or off.
* @param format string
* @return bool
**/
func validFormat(format string) bool {
	return format == FormatText || format == FormatJson || format == FormatOff
}

/**
* writeText: Writes one text line.
* @param line string
**/
func (o *output) writeText(line string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	fmt.Fprintln(o.w, line)
}

/**
* writeJson: Writes the request as one JSON line, with OpenTelemetry attribute names.
* @param s *Metrics
**/
func (o *output) writeJson(s *Metrics) {
	route := s.Route
	if route == "" {
		route = s.Path
	}

	o.json.LogAttrs(context.Background(), o.level(s), "http.request",
		slog.String("http.request.method", s.Method),
		slog.String("http.route", route),
		slog.String("url.path", s.Path),
		slog.String("url.scheme", s.Scheme),
		slog.Int("http.response.status_code", s.StatusCode),
		slog.Float64("duration_ms", s.Latency),
		slog.Float64("response_time_ms", s.ResponseTime),
		slog.Float64("search_time_ms", s.SearchTime),
		slog.Int("http.request.body.size", s.RequestSize),
		slog.Int("http.response.body.size", s.ResponseSize),
		slog.String("client.address", s.ClientAddress),
		slog.String("server.address", s.ServerAddress),
		slog.String("user_agent.original", s.UserAgent),
		slog.String("trace_id", s.TraceID),
		slog.String("service_id", s.ServiceId),
		slog.String("app_name", s.AppName),
	)
}

/**
* level: ERROR for a 5xx, WARN for a 4xx or a latency over LATENCY_LIMIT ms, INFO otherwise.
* @param s *Metrics
* @return slog.Level
**/
func (o *output) level(s *Metrics) slog.Level {
	switch {
	case s.StatusCode >= http.StatusInternalServerError:
		return slog.LevelError
	case s.StatusCode >= http.StatusBadRequest, s.Latency >= o.latencyLimit:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}
