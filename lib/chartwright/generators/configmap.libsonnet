// configmap generator — emits one ConfigMap (or Secret) per entry in the
// component's `configs[]`. Each config's structured value is an opaque,
// install-time-tunable block-string hole: Helm never introspects it, Tempo (or
// whatever) validates it at runtime (§8). Returns an ARRAY of objects; the
// assembler emits one file per entry. Pairs with mounts.libsonnet, which wires
// the volumes/mounts/checksums into the workload from the same configs[] data.
local helm = import '../helm.libsonnet';
local mounts = import '../mounts.libsonnet';

{
  gvk: 'v1/ConfigMap',
  when(c):: std.length(std.get(c, 'configs', [])) > 0,
  build(c):: [
    {
      apiVersion: 'v1',
      kind: std.get(cfg, 'kind', 'ConfigMap'),
      metadata: {
        name: mounts.objectName(c, cfg),
        labels: { 'app.kubernetes.io/name': c.name },
      },
      [if mounts.isSecret(cfg) then 'stringData' else 'data']: {
        [std.get(cfg, 'subPath', cfg.name)]:
          helm.value(c.name + '.configs.' + cfg.name, cfg.value, { render: 'block-string' }),
      },
    }
    for cfg in std.get(c, 'configs', [])
  ],
}
