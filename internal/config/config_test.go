package config

import (
	"os"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	content := `
version = 1
listen_address = "127.0.0.1:3358"
target_host = "127.0.0.1"

domains = [
  "quickproxy.webdemos.online",
  "quickproxy-ws.webdemos.online",
]

log_directory = "/var/log/quickproxy"

minimum_port = 1024
maximum_port = 65535

read_header_timeout = "10s"
idle_timeout = "120s"
shutdown_timeout = "15s"

preserve_original_host = true
trust_forwarded_headers = true
`
	tmpfile, err := os.CreateTemp("", "config-*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.ListenAddress != "127.0.0.1:3358" {
		t.Errorf("ListenAddress = %v, want 127.0.0.1:3358", cfg.ListenAddress)
	}
	if len(cfg.Domains) != 2 {
		t.Errorf("Domains length = %d, want 2", len(cfg.Domains))
	}
	if cfg.MinimumPort != 1024 {
		t.Errorf("MinimumPort = %d, want 1024", cfg.MinimumPort)
	}
	if cfg.MaximumPort != 65535 {
		t.Errorf("MaximumPort = %d, want 65535", cfg.MaximumPort)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "missing domains",
			content: `
listen_address = "127.0.0.1:3358"
log_directory = "/var/log/quickproxy"
minimum_port = 1024
maximum_port = 65535
`,
			wantErr: true,
		},
		{
			name: "invalid port range",
			content: `
listen_address = "127.0.0.1:3358"
domains = ["test.com"]
log_directory = "/var/log/quickproxy"
minimum_port = 70000
maximum_port = 1024
`,
			wantErr: true,
		},
		{
			name: "bad target host",
			content: `
listen_address = "127.0.0.1:3358"
domains = ["test.com"]
log_directory = "/var/log/quickproxy"
minimum_port = 1024
maximum_port = 65535
target_host = "10.0.0.1"
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "config-*.toml")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tt.content)); err != nil {
				t.Fatal(err)
			}
			tmpfile.Close()

			_, err = Load(tmpfile.Name())
			if tt.wantErr && err == nil {
				t.Error("Load() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Load() unexpected error: %v", err)
			}
		})
	}
}
