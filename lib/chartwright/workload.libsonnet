// workload.libsonnet — shared pieces for pod-bearing workloads (Deployment,
// StatefulSet). Keeps selector/labels/pod-template consistent and wires config
// volumes, mounts, and rollout annotations from the component's configs[] (§8).
local helm = import 'helm.libsonnet';
local mounts = import 'mounts.libsonnet';

{
  labels(c):: { 'app.kubernetes.io/name': c.name },
  selector(c):: { matchLabels: $.labels(c) },

  // podTemplate builds the shared spec.template for a component.
  podTemplate(c):: {
    local hasConfigs = std.length(std.get(c, 'configs', [])) > 0,
    local annotations = mounts.checksumAnnotations(c),
    metadata: {
      labels: $.labels(c),
      [if std.length(annotations) > 0 then 'annotations']: annotations,
    },
    spec: {
      containers: [{
        name: c.name,
        image: helm.value(c.name + '.image', std.get(c, 'image', 'busybox:latest'),
                          { render: 'quote' }),
        [if std.length(std.get(c, 'ports', [])) > 0 then 'ports']:
          [{ name: p.name, containerPort: p.port } for p in std.get(c, 'ports', [])],
        [if hasConfigs then 'volumeMounts']: mounts.mounts(c),
      }],
      [if hasConfigs then 'volumes']: mounts.volumes(c),
    },
  },
}
