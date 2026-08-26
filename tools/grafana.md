# Grafana

Grafana is an open-source dashboarding and alerting platform that turns metrics, logs, and traces from many different backends into unified, queryable visualizations — solving the problem of having observability data scattered across a dozen storage systems (Prometheus, Loki, Elasticsearch, CloudWatch, ClickHouse, etc.) with no single place to look at it.

## Primary use cases

- **Operational dashboards.** Real-time views of service health — request rate, error rate, latency (the "RED" metrics), saturation, queue depth — built once and shared across a team.
- **Incident response.** During an outage, engineers pivot from a spiking panel to the underlying logs and traces via linked data sources, without switching tools.
- **Alerting.** Grafana's unified alerting engine evaluates queries against any connected data source and routes notifications to Slack, PagerDuty, Opsgenie, or webhooks — useful when your metrics and logs live in different backends but you want one alerting pipeline.
- **Multi-source correlation.** Because Grafana is data-source agnostic, it's the natural place to overlay business metrics (from a SQL database) against infrastructure metrics (from Prometheus) on the same time axis.

A team typically adopts Grafana once it has more than one telemetry backend (e.g., Prometheus for metrics and Loki or Elasticsearch for logs) and wants a consistent visualization and alerting layer on top, rather than learning each backend's native UI. It's also common as the front end for a self-hosted Prometheus/OTel stack, or as the dashboard layer for a managed observability vendor.

## Basic usage examples

**1. Run Grafana locally with Docker:**

```bash
docker run -d -p 3000:3000 --name=grafana grafana/grafana-oss:latest
# Visit http://localhost:3000 (default login: admin/admin)
```

**2. Add a Prometheus data source via provisioning (for reproducible setup):**

```yaml
# provisioning/datasources/prometheus.yaml
apiVersion: 1
datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
```

**3. Define a dashboard panel as code (JSON model, commonly generated via Grafonnet or Terraform):**

```json
{
  "title": "HTTP Error Rate",
  "type": "timeseries",
  "targets": [
    {
      "expr": "sum(rate(http_requests_total{status=~\"5..\"}[5m])) / sum(rate(http_requests_total[5m]))",
      "refId": "A"
    }
  ]
}
```

**4. Create an alert rule via the Grafana CLI/API using provisioning:**

```yaml
# provisioning/alerting/rules.yaml
apiVersion: 1
groups:
  - orgId: 1
    name: high-error-rate
    folder: SLOs
    interval: 1m
    rules:
      - uid: high-5xx-rate
        title: High 5xx rate
        condition: C
        data:
          - refId: A
            datasourceUid: prometheus
            model:
              expr: sum(rate(http_requests_total{status=~"5.."}[5m]))
        for: 5m
```

## Common pitfalls

- **Dashboards built by hand don't survive.** Clicking through the UI to build a dashboard is fine for exploration, but production dashboards should be defined as JSON/Jsonnet and version-controlled (via provisioning or the Grafana Terraform provider), or they silently drift and get lost when someone "just fixes one panel" in the UI.
- **Expensive queries on shared dashboards.** A dashboard with many panels each running a heavy PromQL/LogQL query, refreshed every 10 seconds, can hammer the underlying data source. Set sensible refresh intervals and use recording rules to precompute expensive aggregations.
- **Alerting on raw queries instead of SLOs.** Alerting directly on noisy raw metrics produces pager fatigue. Prefer burn-rate or threshold alerts derived from a defined SLI/SLO, with appropriate `for:` durations to avoid flapping.
- **Data source permissions are coarse by default.** Without folder/dashboard permissions and data source access control configured, any viewer can query any connected data source — worth locking down before connecting anything with sensitive data (e.g., a production database).
- **Version skew between panels and data source APIs.** Upgrading Grafana or a data source plugin can subtly change query syntax or panel rendering; test dashboard-as-code changes in a staging Grafana instance before rolling to production.
