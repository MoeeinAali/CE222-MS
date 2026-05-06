package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

var (
	mu    sync.Mutex
	l     *log.Logger
	debug bool
)

// Init opens the log file, mirrors to stdout, and reads MINI_NGINX_LOG_LEVEL (info|debug, default info).
func Init(file string) {
	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, f)
	l = log.New(mw, "", log.LstdFlags|log.Lmicroseconds)

	switch strings.ToLower(strings.TrimSpace(os.Getenv("MINI_NGINX_LOG_LEVEL"))) {
	case "debug", "all", "verbose":
		debug = true
	default:
		debug = false
	}
}

func isNoisePath(path string) bool {
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}
	switch strings.ToLower(path) {
	case "/favicon.ico", "/robots.txt", "/static/":
		return true
	default:
		return strings.HasPrefix(path, "/.well-known/")
	}
}

// accessAllowed returns whether an access line should be printed at default (non-debug) level.
func accessAllowed(path string) bool {
	return debug || !isNoisePath(path)
}

// Debug logs only when MINI_NGINX_LOG_LEVEL=debug.
func Debug(msg string, kv ...interface{}) {
	if l == nil || !debug {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	logLine("[DBG]", msg, kv...)
}

// Info logs a non-request message (startup, etc.).
func Info(msg string, kv ...interface{}) {
	if l == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	logLine("[INF]", msg, kv...)
}

// Access logs one line per HTTP request. Noisy paths (favicon, robots) are skipped unless debug.
func Access(remote, method, path, outcome string) {
	if l == nil {
		return
	}
	if !accessAllowed(path) {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	l.Printf("[ACC] remote=%s method=%s path=%s %s\n", remote, method, path, outcome)
}

// Warn logs a warning.
func Warn(msg string, kv ...interface{}) {
	if l == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	logLine("[WRN]", msg, kv...)
}

// Error logs an error; use for non-fatal issues during a request.
func Error(msg string, kv ...interface{}) {
	if l == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	logLine("[ERR]", msg, kv...)
}

func logLine(tag, msg string, kv ...interface{}) {
	if extras := formatPairs(kv...); extras == "" {
		l.Printf("%s %s\n", tag, msg)
	} else {
		l.Printf("%s %s %s\n", tag, msg, extras)
	}
}

// formatPairs turns ("a", 1, "b", "c") into "a=1 b=c" for readable logs.
func formatPairs(kv ...interface{}) string {
	if len(kv) == 0 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < len(kv)-1; i += 2 {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(fmt.Sprintf("%v=%v", kv[i], kv[i+1]))
	}
	if len(kv)%2 == 1 {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(fmt.Sprintf("%v", kv[len(kv)-1]))
	}
	return b.String()
}
