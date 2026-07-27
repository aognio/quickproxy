package proxy

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/gnrfan/quickproxy/internal/config"
	"github.com/gnrfan/quickproxy/internal/hostname"
	"github.com/gnrfan/quickproxy/internal/logging"
)

type Proxy struct {
	config  *config.Config
	logger  *logging.Logger
	log     *slog.Logger
	verbose *slog.Logger
}

func New(cfg *config.Config, logger *logging.Logger, log *slog.Logger) *Proxy {
	return &Proxy{
		config:  cfg,
		logger:  logger,
		log:     log,
		verbose: slog.Default(),
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()

	p.verbose.Debug("incoming request",
		"host", r.Host,
		"method", r.Method,
		"uri", r.RequestURI,
		"remote_addr", r.RemoteAddr,
		"request_id", requestID,
	)

	host := r.Host
	if host == "127.0.0.1" || host == "localhost" || host == p.config.ListenAddress {
		if r.URL.Path == "/healthz" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
			return
		}
	}

	parsed, err := hostname.Parse(host, p.config.Domains)
	if err != nil {
		p.log.Warn("invalid hostname",
			"host", host,
			"error", err.Error(),
			"request_id", requestID,
		)
		writeErrorLog(p.logger, "unknown", 0, host, r.Method, r.URL.RequestURI(), "hostname_parsing", err.Error(), requestID)
		http.Error(w, "400 Bad Request: Invalid hostname", http.StatusBadRequest)
		return
	}

	if parsed.Port < p.config.MinimumPort || parsed.Port > p.config.MaximumPort {
		p.log.Warn("port out of range",
			"app_name", parsed.AppName,
			"port", parsed.Port,
			"range", fmt.Sprintf("%d-%d", p.config.MinimumPort, p.config.MaximumPort),
			"request_id", requestID,
		)
		writeErrorLog(p.logger, parsed.AppName, parsed.Port, host, r.Method, r.URL.RequestURI(), "port_range", fmt.Sprintf("port %d not in allowed range", parsed.Port), requestID)
		http.Error(w, "404 Not Found: Port not allowed", http.StatusNotFound)
		return
	}

	appWriter, err := p.logger.GetWriter(parsed.AppName)
	if err != nil {
		p.log.Error("failed to get log writer",
			"app_name", parsed.AppName,
			"error", err.Error(),
		)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	start := time.Now()

	backendScheme := "http"
	forwardedProto := r.Header.Get("X-Forwarded-Proto")
	if forwardedProto == "" && r.TLS != nil {
		forwardedProto = "https"
	}

	director := func(out *http.Request) {
		out.URL.Scheme = backendScheme
		out.URL.Host = fmt.Sprintf("%s:%d", p.config.TargetHost, parsed.Port)

		out.Header.Set("X-Real-IP", r.RemoteAddr)
		out.Header.Set("X-Forwarded-For", r.RemoteAddr)
		out.Header.Set("X-Forwarded-Host", r.Host)

		if p.config.PreserveOriginalHost {
			out.Header.Set("Host", r.Host)
		} else {
			out.Header.Set("Host", out.URL.Host)
		}

		if forwardedProto != "" {
			out.Header.Set("X-Forwarded-Proto", forwardedProto)
		}

		out.Header.Set("Forwarded", fmt.Sprintf("for=%s;host=%s;proto=%s",
			ipFromAddr(r.RemoteAddr), r.Host, getProto(r)))
	}

	rp := &httputil.ReverseProxy{
		Director:     director,
		Transport:    p.buildTransport(),
		FlushInterval: time.Millisecond * 100,
	}

	wrapped := &statusCapturingResponseWriter{ResponseWriter: w, status: 200}
	rp.ServeHTTP(wrapped, r)

	duration := time.Since(start)
	status := wrapped.Status()

	userAgent := r.Header.Get("User-Agent")
	if userAgent == "" {
		userAgent = "-"
	}

	appWriter.WriteAccess(logging.AccessLogEntry{
		Timestamp:      time.Now(),
		AppName:        parsed.AppName,
		Port:           parsed.Port,
		FrontendHost:   host,
		RemoteAddr:     r.RemoteAddr,
		Method:         r.Method,
		URI:            r.URL.RequestURI(),
		Status:         status,
		Bytes:          0,
		Duration:       duration,
		UserAgent:      userAgent,
		ForwardedProto: forwardedProto,
		RequestID:      requestID,
	})
}

func (p *Proxy) buildTransport() *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout:  1 * time.Second,
	}
}

func writeErrorLog(logger *logging.Logger, appName string, port int, hostname, method, uri, category, message, requestID string) {
	if logger == nil {
		return
	}
	w, err := logger.GetWriter(appName)
	if err != nil {
		return
	}
	w.WriteError(logging.ErrorLogEntry{
		Timestamp: time.Now(),
		AppName:    appName,
		Port:       port,
		Hostname:   hostname,
		Method:     method,
		URI:        uri,
		Category:   category,
		Message:    message,
		RequestID:  requestID,
	})
}

func ipFromAddr(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func getProto(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	return "http"
}

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusCapturingResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusCapturingResponseWriter) Status() int {
	return w.status
}

func generateRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[0] &= 0x7F
	return fmt.Sprintf("%x", b)[:16]
}
