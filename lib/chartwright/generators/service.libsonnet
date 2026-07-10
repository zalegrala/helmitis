// service generator — a ClusterIP Service selecting the component's pods.
{
  gvk: 'v1/Service',
  when(c):: std.length(std.get(c, 'ports', [])) > 0,
  build(c):: {
    apiVersion: 'v1',
    kind: 'Service',
    metadata: {
      name: c.name,
      labels: { 'app.kubernetes.io/name': c.name },
    },
    spec: {
      selector: { 'app.kubernetes.io/name': c.name },
      ports: [{ name: p.name, port: p.port, targetPort: p.name } for p in std.get(c, 'ports', [])],
    },
  },
}
