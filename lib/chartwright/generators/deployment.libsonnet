// deployment generator — a Deployment for a component. Variable points (replicas,
// image) are holes; config volumes/mounts come from workload.podTemplate.
local helm = import '../helm.libsonnet';
local workload = import '../workload.libsonnet';

{
  gvk: 'apps/v1/Deployment',
  build(c):: {
    apiVersion: 'apps/v1',
    kind: 'Deployment',
    metadata: {
      name: c.name,
      labels: workload.labels(c),
    },
    spec: {
      replicas: helm.value(c.name + '.replicas', std.get(c, 'replicas', 1),
                           { schema: { type: 'integer', minimum: 1 } }),
      selector: workload.selector(c),
      template: workload.podTemplate(c),
    },
  },
}
