# Fiscalismia Monitoring

Prometheus & Grafana Dashboard running as a container, ingesting metrics from ingress (Loadbalancer) and egress (NAT-Gateway) servers as well as Demo, Frontend and Backend Instances.

## Architecture Overview

| Domain | Service | Private IP | Description |
|--------|---------|------------|-------------|
| `fiscalismia.com` | Frontend | `172.24.0.3` | Main Fiscalismia Frontend |
| `backend.fiscalismia.com` | Backend | `172.24.0.4` | Main REST API backend |
| `demo.fiscalismia.com` | Demo | `172.20.0.2` | Encapsulated Demo instance (frontend) |
| `backend.demo.fiscalismia.com` | Demo | `172.20.0.2` | Encapsulated Demo instance (backend) |
| `monitoring.fiscalismia.com` | Monitoring | `172.24.0.2` | Prometheus & Grafana dashboard |

## Components

- **Prometheus** - Time series database for metrics collection
- **Grafana** - Visualization and dashboarding
- **Nginx** - TLS terminating reverse proxy (receives proxy-protocol v2 from HAProxy)

## Metrics Sources

### Network Logs (Current Implementation)

| Source | Exporter | Port | Private IP |
|--------|----------|------|------------|
| HAProxy Loadbalancer | Native stats | 8404 | `172.24.1.3` |
| Frontend Nginx | nginx-prometheus-exporter | 9113 | `172.24.0.3` |
| Backend Nginx | nginx-prometheus-exporter | 9113 | `172.24.0.4` |
| Demo Frontend Nginx | nginx-prometheus-exporter | 9113 | `172.20.0.2` |
| Demo Backend Nginx | nginx-prometheus-exporter | 9114 | `172.20.0.2` |
| NAT Gateway Egress | nginx-prometheus-exporter | 9113 | `172.24.1.4` |

### Performance Metrics (Recommended Future Implementation)

For comprehensive performance monitoring, consider implementing these exporters:

| Exporter | Purpose | Recommended For |
|----------|---------|-----------------|
| **node_exporter** | System metrics (CPU, memory, disk, network) | All instances |
| **postgres_exporter** | PostgreSQL database metrics | Database instances |
| **process-exporter** | Per-process resource usage | Backend/Frontend instances |
| **cadvisor** | Container metrics | All containerized services |
| **blackbox_exporter** | HTTP/TCP/ICMP probing | Endpoint health checks |

Best practice for performance monitoring:
1. Deploy `node_exporter` on each host for system-level metrics
2. Use `cadvisor` for container-level resource metrics
3. Add application-specific exporters (e.g., Express.js metrics middleware)
4. Implement `blackbox_exporter` for synthetic monitoring from external perspectives

---

## Deployment

### Prerequisites

1. **TLS Certificates** - Let's Encrypt certificates at:
   ```
   /etc/letsencrypt/live/monitoring.fiscalismia.com/fullchain.pem
   /etc/letsencrypt/live/monitoring.fiscalismia.com/privkey.pem
   ```

2. **Sysctl Configuration** - Allow unprivileged port binding:
   ```bash
   echo "net.ipv4.ip_unprivileged_port_start=80" | sudo tee /etc/sysctl.d/nonroot_can_bind_to_low_ports.conf
   sudo sysctl --system
   ```

3. **nftables Firewall** - Ensure podman networks are allowed:
   - `10.89.0.0/24` | `10.89.1.0/24` | `10.89.2.0/24` | `10.89.3.0/24`

### Files Required on Remote Instance

Copy these files to the monitoring instance deployment directory (e.g., `/opt/fiscalismia-monitoring/`):

```
/opt/fiscalismia-monitoring/
├── docker-compose.yml          # Main compose file
├── .env                        # Environment variables (secrets)
├── Dockerfile.Prometheus       # Prometheus image build
├── Dockerfile.Grafana          # Grafana image build
├── Dockerfile.Nginx            # Nginx reverse proxy build
├── nginx.conf                  # Nginx configuration
├── config/
│   └── prometheus.yml          # Prometheus scrape configuration
├── provisioning/
│   ├── datasources/
│   │   └── datasources.yml     # Grafana datasource config
│   └── dashboards/
│       └── dashboards.yml      # Grafana dashboard provider
└── dashboards/
    └── network-overview.json   # Pre-configured network dashboard
```

### Environment Configuration

1. Copy the template and configure secrets:
   ```bash
   cp .env.template .env
   ```

2. Generate secure passwords and update `.env`:
   ```bash
   # Generate strong password for Grafana admin
   openssl rand -base64 32
   
   # Generate secret key for Grafana
   openssl rand -base64 32
   ```

3. Edit `.env` with actual values:
   ```env
   GRAFANA_ADMIN_USER=admin
   GRAFANA_ADMIN_PASSWORD=<generated-password>
   GRAFANA_SECRET_KEY=<generated-secret-key>
   ```

### Deployment Commands

**Build and Start:**
```bash
# Navigate to deployment directory
cd /opt/fiscalismia-monitoring

# Build images and start services
podman-compose up -d --build

# Or with docker-compose
docker-compose up -d --build
```

**View Logs:**
```bash
podman-compose logs -f

# Individual service logs
podman-compose logs -f fiscalismia-prometheus
podman-compose logs -f fiscalismia-grafana
podman-compose logs -f fiscalismia-monitoring-nginx
```

**Stop Services:**
```bash
podman-compose down
```

**Update Configuration:**
```bash
# After modifying prometheus.yml, reload config without restart
curl -X POST http://127.0.0.1:9090/-/reload

# Or restart the container
podman-compose restart fiscalismia-prometheus
```

**View Container Status:**
```bash
podman-compose ps
```

---

## Network Configuration

### Podman/Docker Networks

| Network | CIDR | Purpose |
|---------|------|---------|
| `monitoring-internal` | `10.89.2.0/24` | Internal communication (Prometheus ↔ Grafana ↔ Nginx) |
| `monitoring-external` | `10.89.3.0/24` | External ingress from HAProxy loadbalancer |

### HAProxy Configuration

Ensure HAProxy is configured to route traffic to the monitoring instance:

```haproxy
frontend https_monitoring
    bind *:443 ssl crt /path/to/cert.pem
    
    acl host_monitoring hdr(host) -i monitoring.fiscalismia.com
    use_backend monitoring_backend if host_monitoring

backend monitoring_backend
    server monitoring 172.24.0.2:443 check send-proxy-v2 ssl verify none
```

---

## Security Features

- **Unprivileged containers** - All services run as non-root users
- **Read-only filesystems** - Containers use read-only root filesystems where possible
- **Dropped capabilities** - All Linux capabilities dropped except required ones
- **No privilege escalation** - `no-new-privileges:true` security option
- **Resource limits** - CPU and memory constraints defined
- **Network isolation** - Internal network has no external access
- **TLS only** - All external traffic encrypted with TLS 1.3
- **Secret management** - Credentials stored in `.env` file (not in images)

---

## Accessing the Dashboard

1. Navigate to: `https://monitoring.fiscalismia.com`
2. Login with credentials from `.env` file
3. Pre-configured dashboards are available under "Fiscalismia" folder
4. Prometheus UI available at: `https://monitoring.fiscalismia.com/prometheus/`

---

## Configuring Remote Instances for Metrics Export

### HAProxy Metrics (Loadbalancer)

Enable stats in HAProxy configuration:
```haproxy
frontend stats
    bind *:8404
    mode http
    http-request use-service prometheus-exporter if { path /metrics }
    stats enable
    stats uri /stats
    stats refresh 10s
```

### Nginx Prometheus Exporter

Deploy nginx-prometheus-exporter sidecar on each nginx instance:

```yaml
# Example docker-compose addition for frontend/backend
nginx-exporter:
  image: nginx/nginx-prometheus-exporter:1.0
  command:
    - '-nginx.scrape-uri=http://127.0.0.1:8080/nginx_status'
  ports:
    - "9113:9113"
```

Ensure nginx has `stub_status` enabled:
```nginx
location /nginx_status {
    stub_status on;
    access_log off;
    allow 127.0.0.1;
    deny all;
}
```

---

## Troubleshooting

### Check Service Health
```bash
# Prometheus health
curl -s http://127.0.0.1:9090/-/healthy

# Grafana health
curl -s http://127.0.0.1:3000/api/health
```

### Verify Prometheus Targets
Access Prometheus UI at `/prometheus/targets` to verify all scrape targets are healthy.

### Container Resource Usage
```bash
podman stats
```

---

## TODO

- [ ] Configure HAProxy stats endpoint on loadbalancer
- [ ] Deploy nginx-prometheus-exporter on frontend/backend instances
- [ ] Add node_exporter to all instances for system metrics
- [ ] Configure alerting rules in Prometheus
- [ ] Set up Alertmanager for notifications
- [ ] Create additional Grafana dashboards for specific services
- [ ] Implement blackbox_exporter for external monitoring

---

## License

MIT License - See [LICENSE](LICENSE) for details.