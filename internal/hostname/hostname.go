package hostname

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

var (
	validAppName = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

type ParsedHostname struct {
	AppName string
	Port    int
	Domain  string
}

func Parse(host string, allowedDomains []string) (*ParsedHostname, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return nil, fmt.Errorf("empty hostname")
	}

	if ip := net.ParseIP(host); ip != nil {
		return nil, fmt.Errorf("IP address not allowed")
	}

	host, _, err := splitHostPort(host)
	if err != nil {
		return nil, fmt.Errorf("invalid hostname: %w", err)
	}

	host = strings.ToLower(host)

	var matchedDomain string
	var remainder string
	for _, domain := range allowedDomains {
		if strings.HasSuffix(host, "."+domain) {
			matchedDomain = domain
			remainder = strings.TrimSuffix(host, "."+domain)
			break
		}
		if host == domain {
			matchedDomain = domain
			remainder = ""
			break
		}
	}

	if matchedDomain == "" {
		return nil, fmt.Errorf("unrecognized domain suffix")
	}

	if remainder == "" {
		return nil, fmt.Errorf("application name is empty")
	}

	parts := strings.Split(remainder, "-")
	if len(parts) < 2 {
		return nil, fmt.Errorf("no port in hostname")
	}

	portStr := parts[len(parts)-1]
	port := 0
	for i := 0; i < len(portStr); i++ {
		if portStr[i] < '0' || portStr[i] > '9' {
			return nil, fmt.Errorf("port must be numeric")
		}
		port = port*10 + int(portStr[i]-'0')
	}

	if port <= 0 {
		return nil, fmt.Errorf("port must be non-zero")
	}

	if port > 65535 {
		return nil, fmt.Errorf("port exceeds maximum")
	}

	appName := strings.Join(parts[:len(parts)-1], "-")

	if appName == "" {
		return nil, fmt.Errorf("application name is empty")
	}

	if !validAppName.MatchString(appName) {
		return nil, fmt.Errorf("invalid application name format")
	}

	return &ParsedHostname{
		AppName: appName,
		Port:    port,
		Domain:  matchedDomain,
	}, nil
}

func splitHostPort(host string) (string, string, error) {
	if strings.Contains(host, "/") {
		return "", "", fmt.Errorf("path not allowed in host")
	}

	for _, c := range host {
		if c < 32 || c == 127 {
			return "", "", fmt.Errorf("control characters not allowed")
		}
	}

	if strings.HasSuffix(host, ".") {
		host = strings.TrimSuffix(host, ".")
	}

	if i := strings.LastIndex(host, ":"); i != -1 {
		before := host[:i]
		after := host[i+1:]
		if strings.Contains(before, ":") {
			return host, "", nil
		}
		return before, after, nil
	}

	return host, "", nil
}

func SanitizeLogName(name string) (string, error) {
	name = strings.ToLower(name)
	name = strings.TrimSpace(name)

	if name == "" {
		return "", fmt.Errorf("empty name")
	}

	clean := make([]rune, 0, len(name))
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			clean = append(clean, r)
		}
	}

	if strings.Contains(name, "..") {
		return "", fmt.Errorf("path traversal attempt")
	}

	result := string(clean)
	if len(result) > 100 {
		result = result[:100]
	}

	if result == "" {
		return "", fmt.Errorf("no valid characters")
	}

	if strings.HasPrefix(result, "-") || strings.HasSuffix(result, "-") {
		return "", fmt.Errorf("cannot start or end with dash")
	}

	return result, nil
}
