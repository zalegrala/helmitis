// deployment generator — a Deployment for a component. Variable points (replicas,
// image) are marked as holes; everything else is stamped at generation time.
local helm = import '../helm.libsonnet';

{
  gvk: 'apps/v1/Deployment',
  build(c):: {
    apiVersion: 'apps/v1',
    kind: 'Deployment',
    metadata: {
      name: c.name,
      labels: { 'app.kubernetes.io/name': c.name },
    },
    spec: {
      replicas: helm.value(c.name + '.replicas', std.get(c, 'replicas', 1),
                           { schema: { type: 'integer', minimum: 1 } }),
      selector: { matchLabels: { 'app.kubernetes.io/name': c.name } },
      template: {
        metadata: { labels: { 'app.kubernetes.io/name': c.name } },
        spec: {
          containers: [{
            name: c.name,
            image: helm.value(c.name + '.image', std.get(c, 'image', 'busybox:latest'),
                              { render: 'quote' }),
            [if std.length(std.get(c, 'ports', [])) > 0 then 'ports']:
              [{ name: p.name, containerPort: p.port } for p in std.get(c, 'ports', [])],
          }],
        },
      },
    },
  },
}
