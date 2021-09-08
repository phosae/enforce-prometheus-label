## enforce Prometheus labels during metric scraping and querying

When we need to implement some monitoring platform for multi-tenant use case, isolation for metrics scraping and
querying is something necessary.

This project provide a demo implementation to achieve this goal, by adding labels to user metrics and promql. By doing this
- metrics push to Prometheus won't mix with each other. We can implement this logic in some mesh/proxy container which forward metrics to Prometheus.
- Also, narrowing Prometheus query from users avoid unauthorized metrics fetching.

note: module dependency `github.com/influxdata/promql/v2` is a replacement of `github.com/prometheus/promql`, as it not
support go module. You can just port code from Prometheus project.

### enforce label matchers to promql

labels to enforce

```shell
app="app"
region="cn"
```

```shell
--- input
container_cpu_usage_seconds_total{app="",namespace="kube-system",container=~".*apiserver.*"}[5m]
container_cpu_usage_seconds_total{app="",namespace="kube-system",container=~".*apiserver.*"}
container_cpu_usage_seconds_total{namespace="kube-system",container=~".*apiserver.*"}[5m]

--- output
 |---- MatrixSelector :: container_cpu_usage_seconds_total{app="app",container=~".*apiserver.*",namespace="kube-system",region="cn"}[5m]
 |---- VectorSelector :: container_cpu_usage_seconds_total{app="app",container=~".*apiserver.*",namespace="kube-system",region="cn"}
 |---- MatrixSelector :: container_cpu_usage_seconds_total{app="app",container=~".*apiserver.*",namespace="kube-system",region="cn"}[5m]
```

### enforce label pairs to metrics

labels to enforce

```shell
app="app"
region="cn"
```

```shell
---input
# HELP promhttp_metric_handler_requests_total Total number of scrapes by HTTP status code.
# TYPE promhttp_metric_handler_requests_total counter
promhttp_metric_handler_requests_total{code="200"} 1000
promhttp_metric_handler_requests_total{code="500"} 2
promhttp_metric_handler_requests_total{code="503"} 1

---output
# HELP promhttp_metric_handler_requests_total Total number of scrapes by HTTP status code.
# TYPE promhttp_metric_handler_requests_total counter
promhttp_metric_handler_requests_total{code="200",app="app",region="cn"} 1000
promhttp_metric_handler_requests_total{code="500",app="app",region="cn"} 2
promhttp_metric_handler_requests_total{code="503",app="app",region="cn"} 1
```