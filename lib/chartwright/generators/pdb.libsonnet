// pdb generator — a PodDisruptionBudget for a component. Applies only when the
// component declares a `pdb` block.
local helm = import '../helm.libsonnet';

{
  gvk: 'policy/v1/PodDisruptionBudget',
  when(c):: std.get(c, 'pdb', null) != null,
  build(c):: {
    apiVersion: 'policy/v1',
    kind: 'PodDisruptionBudget',
    metadata: {
      name: c.name,
      labels: { 'app.kubernetes.io/name': c.name },
    },
    spec: {
      minAvailable: helm.value(c.name + '.pdb.minAvailable', std.get(c.pdb, 'minAvailable', 1),
                               { schema: { type: 'integer', minimum: 0 } }),
      selector: { matchLabels: { 'app.kubernetes.io/name': c.name } },
    },
  },
}
