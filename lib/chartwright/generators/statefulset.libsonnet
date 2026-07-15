// statefulset generator — a StatefulSet for a stateful component. Shares the pod
// template (incl. config volumes/mounts) with the deployment generator.
local helm = import '../helm.libsonnet';
local workload = import '../workload.libsonnet';

{
  gvk: 'apps/v1/StatefulSet',
  build(c):: {
    apiVersion: 'apps/v1',
    kind: 'StatefulSet',
    metadata: {
      name: c.name,
      labels: workload.labels(c),
    },
    spec: {
      serviceName: c.name,
      replicas: helm.value(c.name + '.replicas', std.get(c, 'replicas', 1),
                           { schema: { type: 'integer', minimum: 1 } }),
      selector: workload.selector(c),
      template: workload.podTemplate(c),
    },
  },
}
