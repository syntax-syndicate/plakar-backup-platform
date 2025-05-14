package logging

import (
	"io"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

type Logger struct {
	EnabledInfo       bool
	EnabledTracing    string
	mutraceSubsystems sync.Mutex
	traceSubsystems   map[string]bool
	stdoutLogger      *log.Logger
	stderrLogger      *log.Logger
	infoLogger        *log.Logger
	warnLogger        *log.Logger
	errorLogger       *log.Logger
	debugLogger       *log.Logger
	traceLogger       *log.Logger
}

func NewLogger(stdout io.Writer, stderr io.Writer) *Logger {
	return &Logger{
		EnabledInfo:     false,
		EnabledTracing:  "",
		stdoutLogger:    log.NewWithOptions(stdout, log.Options{}),
		stderrLogger:    log.NewWithOptions(stderr, log.Options{}),
		infoLogger:      log.NewWithOptions(stdout, log.Options{Level: log.InfoLevel, Prefix: "info", TimeFormat: time.RFC3339}),
		warnLogger:      log.NewWithOptions(stderr, log.Options{Level: log.WarnLevel, Prefix: "warn", TimeFormat: time.RFC3339}),
		debugLogger:     log.NewWithOptions(stdout, log.Options{Level: log.DebugLevel, Prefix: "debug", TimeFormat: time.RFC3339}),
		traceLogger:     log.NewWithOptions(stdout, log.Options{Level: log.DebugLevel, Prefix: "trace", TimeFormat: time.RFC3339}),
		errorLogger:     log.NewWithOptions(stderr, log.Options{Level: log.ErrorLevel, Prefix: "error", TimeFormat: time.RFC3339}),
		traceSubsystems: make(map[string]bool),
	}
}

func (l *Logger) SetOutput(w io.Writer) {
	l.stdoutLogger.SetOutput(w)
	l.stderrLogger.SetOutput(w)
	l.infoLogger.SetOutput(w)
	l.warnLogger.SetOutput(w)
	l.errorLogger.SetOutput(w)
	l.debugLogger.SetOutput(w)
	l.traceLogger.SetOutput(w)
}

func (l *Logger) SetSyslogOutput(w io.Writer) {
	l.stdoutLogger = log.NewWithOptions(w, log.Options{Prefix: "stdout"})
	l.stderrLogger = log.NewWithOptions(w, log.Options{Prefix: "stderr"})
	l.infoLogger = log.NewWithOptions(w, log.Options{Level: log.InfoLevel, Prefix: "info"})
	l.warnLogger = log.NewWithOptions(w, log.Options{Level: log.WarnLevel, Prefix: "warn"})
	l.debugLogger = log.NewWithOptions(w, log.Options{Level: log.DebugLevel, Prefix: "debug"})
	l.traceLogger = log.NewWithOptions(w, log.Options{Level: log.DebugLevel, Prefix: "trace"})
	l.errorLogger = log.NewWithOptions(w, log.Options{Level: log.ErrorLevel, Prefix: "error"})
}

func (l *Logger) Printf(format string, args ...interface{}) {
	l.infoLogger.Printf(format, args...)
}

func (l *Logger) Stdout(format string, args ...interface{}) {
	l.stdoutLogger.Printf(format, args...)
}

func (l *Logger) Stderr(format string, args ...interface{}) {
	l.stderrLogger.Printf(format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	if l.EnabledInfo {
		l.infoLogger.Printf(format, args...)
	}
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.warnLogger.Printf(format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.errorLogger.Printf(format, args...)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.debugLogger.Printf(format, args...)
}

func (l *Logger) Trace(subsystem string, format string, args ...interface{}) {
	if l.EnabledTracing != "" {
		l.mutraceSubsystems.Lock()
		_, exists := l.traceSubsystems[subsystem]
		if !exists {
			_, exists = l.traceSubsystems["all"]
		}
		l.mutraceSubsystems.Unlock()
		if exists {
			l.traceLogger.Printf(subsystem+": "+format, args...)
		}
	}
}

func (l *Logger) EnableInfo() {
	l.EnabledInfo = true
}

func (l *Logger) EnableTracing(traces string) {
	l.EnabledTracing = traces
	l.traceSubsystems = make(map[string]bool)
	for _, subsystem := range strings.Split(traces, ",") {
		l.traceSubsystems[subsystem] = true
	}
}
