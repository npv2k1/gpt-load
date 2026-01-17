# Prometheus Monitoring

GPT-Load now includes built-in Prometheus metrics for comprehensive monitoring and observability.

## Metrics Endpoint

The Prometheus metrics are exposed at:

```
http://localhost:3001/metrics
```

This endpoint returns metrics in the standard Prometheus text format and can be scraped by any Prometheus server.

## Available Metrics

### HTTP Metrics

Standard HTTP request metrics collected for all endpoints:

- **`http_requests_total`** (Counter)
  - Total number of HTTP requests
  - Labels: `method`, `endpoint`, `status`

- **`http_request_duration_seconds`** (Histogram)
  - HTTP request duration in seconds
  - Labels: `method`, `endpoint`, `status`
  - Buckets: [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]

- **`http_request_size_bytes`** (Histogram)
  - HTTP request size in bytes
  - Labels: `method`, `endpoint`
  - Buckets: [100, 1000, 10000, 100000, 1000000, ...]

- **`http_response_size_bytes`** (Histogram)
  - HTTP response size in bytes
  - Labels: `method`, `endpoint`
  - Buckets: [100, 1000, 10000, 100000, 1000000, ...]

### Application-Specific Metrics

Metrics specific to GPT-Load's key management and proxy functionality:

- **`gpt_load_active_keys_total`** (Gauge)
  - Total number of active API keys per group
  - Labels: `group`

- **`gpt_load_invalid_keys_total`** (Gauge)
  - Total number of invalid API keys per group
  - Labels: `group`

- **`gpt_load_proxy_requests_total`** (Counter)
  - Total number of proxy requests per group
  - Labels: `group`, `status`

- **`gpt_load_proxy_request_duration_seconds`** (Histogram)
  - Proxy request duration in seconds
  - Labels: `group`
  - Buckets: [0.1, 0.5, 1, 2, 5, 10, 30, 60, 120]

- **`gpt_load_key_rotations_total`** (Counter)
  - Total number of key rotations per group
  - Labels: `group`

- **`gpt_load_key_validation_total`** (Counter)
  - Total number of key validations
  - Labels: `group`, `result`

## Prometheus Configuration

To scrape metrics from GPT-Load, add the following job to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'gpt-load'
    static_configs:
      - targets: ['localhost:3001']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

## Example Queries

Here are some useful PromQL queries for monitoring GPT-Load:

### HTTP Request Rate
```promql
rate(http_requests_total[5m])
```

### Average Request Duration
```promql
rate(http_request_duration_seconds_sum[5m]) / rate(http_request_duration_seconds_count[5m])
```

### 95th Percentile Request Duration
```promql
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

### Proxy Request Success Rate
```promql
sum(rate(gpt_load_proxy_requests_total{status="success"}[5m])) / sum(rate(gpt_load_proxy_requests_total[5m]))
```

### Active vs Invalid Keys Ratio
```promql
gpt_load_active_keys_total / (gpt_load_active_keys_total + gpt_load_invalid_keys_total)
```

## Grafana Dashboard

You can create a Grafana dashboard to visualize these metrics. Import the following panels:

1. **Request Rate** - Graph showing `rate(http_requests_total[5m])`
2. **Response Time** - Graph showing request duration percentiles
3. **Error Rate** - Graph showing error responses over time
4. **Active Keys** - Gauge showing `gpt_load_active_keys_total`
5. **Proxy Performance** - Graph showing proxy request duration by group

## Integration with Alertmanager

Example alert rules for GPT-Load:

```yaml
groups:
  - name: gpt-load
    rules:
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} errors/sec"

      - alert: NoActiveKeys
        expr: gpt_load_active_keys_total == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "No active keys available"
          description: "Group {{ $labels.group }} has no active keys"

      - alert: HighInvalidKeyRate
        expr: gpt_load_invalid_keys_total / gpt_load_active_keys_total > 0.5
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High rate of invalid keys"
          description: "Group {{ $labels.group }} has {{ $value }}% invalid keys"
```

## Security Considerations

The `/metrics` endpoint is publicly accessible by default. If you need to secure it:

1. Use a reverse proxy (nginx, Traefik) to restrict access
2. Implement network-level access controls
3. Use Prometheus authentication features

Example nginx configuration:

```nginx
location /metrics {
    allow 192.168.1.0/24;  # Prometheus server network
    deny all;
    proxy_pass http://localhost:3001/metrics;
}
```
