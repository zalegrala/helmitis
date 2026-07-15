// servicemonitor generator — a Prometheus-operator ServiceMonitor scraping the
// component's Service ports. Applies when the component opts in with
// `serviceMonitor: true` and exposes ports.
{
  gvk: 'monitoring.coreos.com/v1/ServiceMonitor',
  when(c):: std.get(c, 'serviceMonitor', false) && std.length(std.get(c, 'ports', [])) > 0,
  build(c):: {
    apiVersion: 'monitoring.coreos.com/v1',
    kind: 'ServiceMonitor',
    metadata: {
      name: c.name,
      labels: { 'app.kubernetes.io/name': c.name },
    },
    spec: {
      selector: { matchLabels: { 'app.kubernetes.io/name': c.name } },
      endpoints: [{ port: p.name } for p in std.get(c, 'ports', [])],
    },
  },
}
