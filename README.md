# HyprKnot

A lightweight, efficient, and reliable HTTP API wrapper for KnotDNS 3.4.6+. Designed specifically for infrastructure providers who need to give customers programmatic access to DNS records, particularly PTR records for virtual machines.

## 🚀 Features

- **Lightweight & Fast**: Single binary deployment with minimal resource usage
- **Production Ready**: Comprehensive error handling, logging, and monitoring
- **Secure by Default**: API key authentication, rate limiting, and security headers
- **KnotDNS 3.4.6+ Compatible**: Direct integration via `knotc` commands
- **RESTful API**: Clean, intuitive API design
- **Systemd Integration**: Ready for production deployment as a system service
- **Zone Restrictions**: Configurable zone access control
- **Comprehensive Logging**: Structured JSON logging with multiple output options

## 📋 Supported Record Types

- **A** - IPv4 address records
- **AAAA** - IPv6 address records  
- **PTR** - Reverse DNS pointer records (perfect for VM infrastructure)
- **CNAME** - Canonical name records
- **MX** - Mail exchange records
- **TXT** - Text records
- **NS** - Name server records

## 🛠 Installation

### Prerequisites

- KnotDNS 3.4.6 or later installed and configured
- Go 1.21+ (for building from source)
- Linux system with systemd (for service deployment)

### Quick Install

```bash
# Clone the repository
git clone https://github.com/hyprknot/hyprknot.git
cd hyprknot

# Build and install
make install

# Configure
sudo nano /etc/hyprknot/config.yaml

# Start the service
sudo systemctl enable --now hyprknot
```

### Manual Build

```bash
# Build the binary
make build

# Run locally
./build/hyprknot -config config.yaml
```

## ⚙️ Configuration

Edit `/etc/hyprknot/config.yaml`:

```yaml
# Server configuration
server:
  host: "127.0.0.1"
  port: 8080

# KnotDNS configuration
knot:
  knotc_path: "/usr/bin/knotc"
  socket_path: "/run/knot/knot.sock"
  allowed_zones:
    - "yourdomain.com"
    - "10.in-addr.arpa"  # For PTR records

# Authentication
auth:
  enabled: true
  api_keys:
    - "your-secure-api-key-here"

# Logging
log:
  level: "info"
  format: "json"
  output: "stdout"
```

## 🔌 API Usage

### Authentication

Include your API key in requests:

```bash
# Using X-API-Key header
curl -H "X-API-Key: your-api-key" http://localhost:8080/api/v1/zones

# Using Authorization header
curl -H "Authorization: Bearer your-api-key" http://localhost:8080/api/v1/zones
```

### Endpoints

#### Health Check
```bash
GET /health
```

#### List Zones
```bash
GET /api/v1/zones
```

#### List Records in Zone
```bash
GET /api/v1/zones/example.com/records
```

#### Get Specific Record
```bash
GET /api/v1/zones/example.com/records/host/A
```

#### Create PTR Record (Perfect for VMs)
```bash
POST /api/v1/zones/143.31.194.in-addr.arpa/records
Content-Type: application/json

{
  "name": "100.143.31.194.in-addr.arpa",
  "type": "PTR",
  "ttl": 900,
  "data": "vm-customer-1.hypr.tech."
}
```

#### Create A Record
```bash
POST /api/v1/zones/example.com/records
Content-Type: application/json

{
  "name": "vm-customer-1",
  "type": "A", 
  "ttl": 300,
  "data": "10.0.0.100"
}
```

#### Update Record
```bash
PUT /api/v1/zones/example.com/records/vm-customer-1/A
Content-Type: application/json

{
  "ttl": 600,
  "data": "10.0.0.101"
}
```

#### Delete Record
```bash
DELETE /api/v1/zones/example.com/records/vm-customer-1/A
```

#### Reload Zone
```bash
POST /api/v1/zones/example.com/reload
```

## 🏗 Infrastructure Use Case

Perfect for VM hosting providers:

```bash
# When provisioning a new VM at 194.31.143.100 for customer "acme"

# 1. Create A record
curl -X POST -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name":"vm-acme","type":"A","ttl":900,"data":"194.31.143.100"}' \
  http://100.100.10.80:8080/api/v1/zones/hypr.tech/records

# 2. Create PTR record
curl -X POST -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name":"100.143.31.194.in-addr.arpa","type":"PTR","ttl":900,"data":"vm-acme.hypr.tech."}' \
  http://100.100.10.80:8080/api/v1/zones/143.31.194.in-addr.arpa/records
```

## 🔒 Security

- **API Key Authentication**: Secure access control
- **Zone Restrictions**: Limit access to specific zones
- **Rate Limiting**: Prevent API abuse
- **Security Headers**: OWASP recommended headers
- **Input Validation**: Comprehensive request validation
- **Audit Logging**: All operations are logged

## 📊 Monitoring

### Health Check
```bash
curl http://localhost:8080/health
```

### Logs
```bash
# View logs
sudo journalctl -u hyprknot -f

# Log format (JSON)
{
  "level": "info",
  "msg": "Created record vm-test A in zone example.com",
  "time": "2024-01-15T10:30:45Z"
}
```

## 🚀 Deployment

### Systemd Service

```bash
# Install and start
make install
sudo systemctl enable --now hyprknot

# Check status
sudo systemctl status hyprknot

# View logs
sudo journalctl -u hyprknot -f
```

### Docker (Optional)

```dockerfile
FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY hyprknot /usr/local/bin/
COPY config.yaml /etc/hyprknot/
EXPOSE 8080
CMD ["hyprknot", "-config", "/etc/hyprknot/config.yaml"]
```

## 🛠 Development

```bash
# Run tests
make test

# Format code
make fmt

# Lint code  
make lint

# Run locally
make run

# Build for multiple architectures
make build-all
```

## 📝 License

MIT License - see LICENSE file for details.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📞 Support

- GitHub Issues: Report bugs and feature requests
- Documentation: Check the `/api/v1/docs` endpoint
- Community: Join discussions in GitHub Discussions

## 🎯 Roadmap

- [ ] Web UI dashboard
- [ ] Bulk operations API
- [ ] DNSSEC support
- [ ] Metrics/Prometheus integration
- [ ] Multi-server support
- [ ] Webhook notifications

---

**Built for infrastructure providers who need reliable, lightweight DNS management.**
