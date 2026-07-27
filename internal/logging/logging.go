package logging

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Logger struct {
	mu       sync.Mutex
	writers  map[string]*appWriter
	logDir   string
}

type appWriter struct {
	access *os.File
	error_ *os.File
	accBuf *bufio.Writer
	errBuf *bufio.Writer
	mu     sync.Mutex
}

func New(logDir string) (*Logger, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("creating log directory: %w", err)
	}

	return &Logger{
		writers: make(map[string]*appWriter),
		logDir:  logDir,
	}, nil
}

func (l *Logger) GetWriter(appName string) (*appWriter, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if w, ok := l.writers[appName]; ok {
		return w, nil
	}

	accPath := filepath.Join(l.logDir, appName+"-access.log")
	errPath := filepath.Join(l.logDir, appName+"-error.log")

	accFile, err := os.OpenFile(accPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return nil, fmt.Errorf("opening access log: %w", err)
	}

	errFile, err := os.OpenFile(errPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		accFile.Close()
		return nil, fmt.Errorf("opening error log: %w", err)
	}

	w := &appWriter{
		access: accFile,
		error_: errFile,
		accBuf: bufio.NewWriter(accFile),
		errBuf: bufio.NewWriter(errFile),
	}

	l.writers[appName] = w
	return w, nil
}

func (w *appWriter) WriteAccess(entry AccessLogEntry) {
	w.mu.Lock()
	defer w.mu.Unlock()

	fmt.Fprintf(w.accBuf, "%s %s %d %s %s %s %s %d %d %v %s %s %s\n",
		entry.Timestamp.Format(time.RFC3339),
		entry.AppName,
		entry.Port,
		entry.FrontendHost,
		entry.RemoteAddr,
		entry.Method,
		entry.URI,
		entry.Status,
		entry.Bytes,
		entry.Duration,
		entry.UserAgent,
		entry.ForwardedProto,
		entry.RequestID,
	)
	w.accBuf.Flush()
}

func (w *appWriter) WriteError(entry ErrorLogEntry) {
	w.mu.Lock()
	defer w.mu.Unlock()

	fmt.Fprintf(w.errBuf, "%s %s %d %s %s %s %s %s\n",
		entry.Timestamp.Format(time.RFC3339),
		entry.AppName,
		entry.Port,
		entry.Hostname,
		entry.Method,
		entry.URI,
		entry.Category,
		entry.Message,
	)
	w.errBuf.Flush()
}

func (w *appWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.accBuf.Flush()
	w.errBuf.Flush()

	var errs []error
	if err := w.access.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := w.error_.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var errs []error
	for _, w := range l.writers {
		if err := w.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

type AccessLogEntry struct {
	Timestamp      time.Time
	AppName        string
	Port           int
	FrontendHost   string
	RemoteAddr     string
	Method         string
	URI            string
	Status         int
	Bytes          int64
	Duration       time.Duration
	UserAgent      string
	ForwardedProto string
	RequestID      string
}

type ErrorLogEntry struct {
	Timestamp time.Time
	AppName   string
	Port      int
	Hostname  string
	Method    string
	URI       string
	Category  string
	Message   string
	RequestID string
}
