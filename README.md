# QuickProxy

Dynamic reverse proxy daemon that eliminates the need for individual Nginx virtual hosts for temporary local web applications.

## Purpose

QuickProxy receives wildcard domain traffic from Nginx and dynamically reverse-proxies requests to local TCP ports based on the hostname pattern: `<app-name>-<port>.<domain>`.

## Hostname Grammar

```
<application-name>-<port>.<configured-domain>
```

- **application-name**: lowercase letters, numbers, and dashes (`[a-z0-9-]+`)
- **port**: numeric port number (1024-65535 by default)
- **configured-domain**: one of the configured domain suffixes

## Examples

| Hostname | Target |
|----------|--------|
| `demo-8080.quickproxy.webdemos.online` | `http://127.0.0.1:8080` |
| `my-cool-app-3000.quickproxy.webdemos.online` | `http://127.0.0.1:3000` |
| `vite-5173.quickproxy-ws.webdemos.online` | `http://127.0.0.1:5173` |

## Configuration

Main config: `/etc/quickproxy/server-config.toml`

Key settings:
- `listen_address`: Local address for QuickProxy (default: `127.0.0.1:3358`)
- `target_host`: Backend target host (default: `127.0.0.1`)
- `domains`: List of accepted domain suffixes
- `minimum_port` / `maximum_port`: Allowed backend port range
- `log_directory`: Where per-app logs are written

## Installation

```bash
# Build
go build -o quickproxyd ./cmd/quickproxyd

# Install binary
sudo install -o root -g root -m 0755 quickproxyd /usr/local/bin/quickproxyd

# Create system user
sudo useradd -r -s /usr/sbin/nologin quickproxy

# Create directories
sudo mkdir -p /etc/quickproxy /var/log/quickproxy
sudo chown quickproxy:quickproxy /var/log/quickproxy

# Install config
sudo install -o root -g quickproxy -m 0640 deploy/examples/server-config.toml /etc/quickproxy/server-config.toml

# Install systemd unit
sudo install -o root -g root -m 0644 deploy/systemd/quickproxyd.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable quickproxyd
```

## Nginx Architecture

Two wildcard Nginx virtual hosts forward to the same `quickproxyd` process:

1. **`quickproxy.webdemos.online`** - Regular HTTP traffic
2. **`quickproxy-ws.webdemos.online`** - WebSocket-enabled traffic

Both use the same backend (`http://127.0.0.1:3358`). The distinction is an Nginx ingress policy:
- `quickproxy`: No WebSocket upgrade headers forwarded
- `quickproxy-ws`: Nginx `map` directive enables WebSocket upgrade when the client sends `Upgrade` headers

## Log Locations

Per-application logs in `/var/log/quickproxy/`:
- `<app-name>-access.log`
- `<app-name>-error.log`

## Health Endpoint

```bash
curl -H "Host: 127.0.0.1" http://127.0.0.1:3358/healthz
# {"status":"ok"}
```

## Doctor Command

Diagnostic checks for installation verification:

```bash
quickproxyd --doctor
quickproxyd --doctor --doctor-format json
```

## Troubleshooting

### Test with curl using --resolve

```bash
curl --resolve myapp-8080.quickproxy.webdemos.online:443:127.0.0.1 \
  https://myapp-8080.quickproxy.webdemos.online/
```

### Check service status

```bash
sudo systemctl status quickproxyd
sudo journalctl -u quickproxyd --no-pager -n 50
```

### Common issues

- **502 Bad Gateway**: Backend port not listening or firewalled
- **400 Bad Request**: Malformed hostname or wrong domain suffix
- **404 Not Found**: Port outside allowed range

## Security Constraints

- Only accepts connections on loopback (`127.0.0.1`)
- Only proxies to `127.0.0.1` or `localhost`
- Backend ports restricted to configurable range (default 1024-65535)
- No arbitrary external proxying
- Strict hostname validation
