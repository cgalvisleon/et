package jwf

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
)

// MAX_LOG_MESSAGE: the longest message of a console line sent to OnLog (the server log keeps it whole)
const MAX_LOG_MESSAGE = 4096

/**
* consoleOf: The console of a script: each call goes to the server log (after prefix) and, with emit, as a line
* {level, message, at} (what OnLog of a step or an instance receives)
* @param prefix string, emit func(line et.Json)
* @return map[string]any
**/
func consoleOf(prefix string, emit func(line et.Json)) map[string]any {
	write := func(level string, args []any) string {
		message := messageOf(args)
		if emit != nil {
			emit(et.Json{"level": level, "message": truncate(message), "at": timezone.Now()})
		}
		if prefix == "" {
			return message
		}
		return fmt.Sprintf("%s - %s", prefix, message)
	}

	return map[string]any{
		"log": func(args ...any) {
			logs.Log("LOG", write("log", args))
		},
		"debug": func(args ...any) {
			logs.Debug(write("debug", args))
		},
		"info": func(args ...any) {
			logs.Info(write("info", args))
		},
		"warn": func(args ...any) {
			logs.Warn(write("warn", args))
		},
		"error": func(args ...any) {
			logs.Error(errors.New(write("error", args)))
		},
	}
}

/**
* messageOf: The arguments of a console call as one text: strings as they are, the rest as json (an object, params)
* @param args []any
* @return string
**/
func messageOf(args []any) string {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		if text, ok := arg.(string); ok {
			parts = append(parts, text)
			continue
		}
		bt, err := json.Marshal(arg)
		if err != nil {
			parts = append(parts, fmt.Sprint(arg))
			continue
		}
		parts = append(parts, string(bt))
	}
	return strings.Join(parts, " ")
}

/**
* truncate: The message cut at MAX_LOG_MESSAGE
* @param message string
* @return string
**/
func truncate(message string) string {
	if len(message) <= MAX_LOG_MESSAGE {
		return message
	}
	return message[:MAX_LOG_MESSAGE] + "…"
}
