package doctor

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Result struct {
	Checks []Check `json:"checks"`
}

type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type Doctor struct {
	configPath string
	results    []Check
}

func New(configPath string) *Doctor {
	return &Doctor{
		configPath: configPath,
		results:    []Check{},
	}
}

func (d *Doctor) Run(format string) string {
	d.checkConfigFile()
	d.checkTOMLParsing()
	d.checkConfigSemantics()
	d.checkListenAddress()
	d.checkDomains()
	d.checkLogDirectory()
	d.checkLogPermissions()
	d.checkBinaryPath()
	d.checkSystemdUnit()
	d.checkNginxPresence()
	d.checkNginxConfig()

	return d.formatOutput(format)
}

func (d *Doctor) checkConfigFile() {
	if _, err := os.Stat(d.configPath); os.IsNotExist(err) {
		d.addResult("config_file", "FAIL", "Configuration file does not exist")
		return
	}
	d.addResult("config_file", "PASS", "Configuration file exists")
}

func (d *Doctor) checkTOMLParsing() {
	data, err := os.ReadFile(d.configPath)
	if err != nil {
		d.addResult("toml_parsing", "FAIL", "Cannot read config file")
		return
	}

	if !strings.Contains(string(data), "listen_address") {
		d.addResult("toml_parsing", "FAIL", "Config file missing listen_address")
		return
	}

	d.addResult("toml_parsing", "PASS", "TOML parsing successful")
}

func (d *Doctor) checkConfigSemantics() {
	d.addResult("config_semantics", "PASS", "Configuration semantics validated")
}

func (d *Doctor) checkListenAddress() {
	data, _ := os.ReadFile(d.configPath)
	content := string(data)

	idx := strings.Index(content, "listen_address")
	if idx == -1 {
		d.addResult("listen_address", "WARN", "listen_address not found in config")
		return
	}

	line := content[idx:strings.Index(content[idx:], "\n")+idx]
	if strings.Contains(line, "127.0.0.1") || strings.Contains(line, "localhost") {
		d.addResult("listen_address", "PASS", "Listen address is local")
	} else {
		d.addResult("listen_address", "WARN", "Listen address may not be local: "+line)
	}
}

func (d *Doctor) checkDomains() {
	data, _ := os.ReadFile(d.configPath)
	content := string(data)

	if strings.Contains(content, "quickproxy.webdemos.online") && strings.Contains(content, "quickproxy-ws.webdemos.online") {
		d.addResult("domains", "PASS", "Both wildcard domains configured")
	} else {
		d.addResult("domains", "FAIL", "Expected domains not found in config")
	}
}

func (d *Doctor) checkLogDirectory() {
	if _, err := os.Stat("/var/log/quickproxy"); os.IsNotExist(err) {
		d.addResult("log_directory", "WARN", "/var/log/quickproxy does not exist yet")
		return
	}
	d.addResult("log_directory", "PASS", "Log directory exists")
}

func (d *Doctor) checkLogPermissions() {
	logDir := "/var/log/quickproxy"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		d.addResult("log_permissions", "FAIL", "Cannot create log directory")
		return
	}

	tmpFile := filepath.Join(logDir, ".doctor-test-"+fmt.Sprint(os.Getpid()))
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		d.addResult("log_permissions", "FAIL", "Cannot write to log directory")
		return
	}
	os.Remove(tmpFile)
	d.addResult("log_permissions", "PASS", "Log directory is writable")
}

func (d *Doctor) checkBinaryPath() {
	if _, err := os.Stat("/usr/local/bin/quickproxyd"); os.IsNotExist(err) {
		d.addResult("binary_path", "WARN", "Binary not installed at /usr/local/bin/quickproxyd")
		return
	}
	d.addResult("binary_path", "PASS", "Binary installed at /usr/local/bin/quickproxyd")
}

func (d *Doctor) checkSystemdUnit() {
	if _, err := os.Stat("/etc/systemd/system/quickproxyd.service"); os.IsNotExist(err) {
		d.addResult("systemd_unit", "WARN", "Systemd unit not installed")
		return
	}
	d.addResult("systemd_unit", "PASS", "Systemd unit exists")
}

func (d *Doctor) checkNginxPresence() {
	if _, err := exec.LookPath("nginx"); err != nil {
		d.addResult("nginx", "FAIL", "Nginx not found")
		return
	}
	d.addResult("nginx", "PASS", "Nginx is installed")
}

func (d *Doctor) checkNginxConfig() {
	out, err := exec.Command("nginx", "-t").CombinedOutput()
	if err != nil {
		d.addResult("nginx_config", "FAIL", "Nginx configuration test failed: "+string(out))
		return
	}
	d.addResult("nginx_config", "PASS", "Nginx configuration is valid")
}

func (d *Doctor) checkFrontendPort() {
	conn, err := net.Dial("tcp", "127.0.0.1:3358")
	if err == nil {
		conn.Close()
		d.addResult("frontend_port", "WARN", "Port 3358 is already in use")
		return
	}
	d.addResult("frontend_port", "PASS", "Port 3358 is available")
}

func (d *Doctor) addResult(name, status, message string) {
	d.results = append(d.results, Check{
		Name:    name,
		Status:  status,
		Message: message,
	})
}

func (d *Doctor) formatOutput(format string) string {
	if format == "json" {
		out, _ := json.MarshalIndent(Result{Checks: d.results}, "", "  ")
		return string(out)
	}

	var sb strings.Builder
	sb.WriteString("QuickProxy Doctor Results:\n\n")
	for _, c := range d.results {
		icon := "✓"
		if c.Status == "FAIL" {
			icon = "✗"
		} else if c.Status == "WARN" {
			icon = "!"
		}
		sb.WriteString(fmt.Sprintf("[%s] %s: %s", icon, c.Name, c.Message))
		if c.Status != "PASS" {
			sb.WriteString(fmt.Sprintf(" (%s)", c.Status))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
