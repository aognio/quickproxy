package hostname

import (
	"testing"
)

func TestParseSimpleHostname(t *testing.T) {
	domains := []string{"quickproxy.webdemos.online", "quickproxy-ws.webdemos.online"}

	tests := []struct {
		name      string
		host      string
		wantApp   string
		wantPort  int
		wantDomain string
		wantErr   bool
	}{
		{
			name:       "simple app with port",
			host:       "myapp-8080.quickproxy.webdemos.online",
			wantApp:    "myapp",
			wantPort:   8080,
			wantDomain: "quickproxy.webdemos.online",
			wantErr:    false,
		},
		{
			name:       "multi-dash app name",
			host:       "my-cool-nodejs-app-7890.quickproxy.webdemos.online",
			wantApp:    "my-cool-nodejs-app",
			wantPort:   7890,
			wantDomain: "quickproxy.webdemos.online",
			wantErr:    false,
		},
		{
			name:       "ws domain",
			host:       "vite-dashboard-5173.quickproxy-ws.webdemos.online",
			wantApp:    "vite-dashboard",
			wantPort:   5173,
			wantDomain: "quickproxy-ws.webdemos.online",
			wantErr:    false,
		},
		{
			name:       "with port in host header",
			host:       "demo-5333.quickproxy.webdemos.online:443",
			wantApp:    "demo",
			wantPort:   5333,
			wantDomain: "quickproxy.webdemos.online",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.host, domains)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Parse() unexpected error: %v", err)
				return
			}
			if got.AppName != tt.wantApp {
				t.Errorf("Parse().AppName = %v, want %v", got.AppName, tt.wantApp)
			}
			if got.Port != tt.wantPort {
				t.Errorf("Parse().Port = %v, want %v", got.Port, tt.wantPort)
			}
			if got.Domain != tt.wantDomain {
				t.Errorf("Parse().Domain = %v, want %v", got.Domain, tt.wantDomain)
			}
		})
	}
}

func TestParseRejection(t *testing.T) {
	domains := []string{"quickproxy.webdemos.online", "quickproxy-ws.webdemos.online"}

	tests := []struct {
		name string
		host string
	}{
		{name: "empty hostname", host: ""},
		{name: "empty app name", host: "-8080.quickproxy.webdemos.online"},
		{name: "ends with dash", host: "app--8080.quickproxy.webdemos.online"},
		{name: "starts with dash", host: "-app-8080.quickproxy.webdemos.online"},
		{name: "port zero", host: "app-0.quickproxy.webdemos.online"},
		{name: "port too high", host: "app-70000.quickproxy.webdemos.online"},
		{name: "non-numeric port", host: "app-abc.quickproxy.webdemos.online"},
		{name: "unrelated domain", host: "app-8080.example.com"},
		{name: "no domain", host: "app-8080"},
		{name: "no port", host: "app.quickproxy.webdemos.online"},
		{name: "path injection", host: "app-8080.quickproxy.webdemos.online/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.host, domains)
			if err == nil {
				t.Errorf("Parse(%q) expected error, got nil", tt.host)
			}
		})
	}
}

func TestSanitizeLogName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "simple name", input: "myapp", want: "myapp", wantErr: false},
		{name: "with dashes", input: "my-cool-app", want: "my-cool-app", wantErr: false},
		{name: "uppercase normalized", input: "MyApp", want: "myapp", wantErr: false},
		{name: "empty", input: "", want: "", wantErr: true},
		{name: "only special chars", input: "---", want: "", wantErr: true},
		{name: "path traversal", input: "../etc/passwd", want: "", wantErr: true},
		{name: "with underscores", input: "my_app", want: "my_app", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SanitizeLogName(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("SanitizeLogName(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("SanitizeLogName(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("SanitizeLogName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
