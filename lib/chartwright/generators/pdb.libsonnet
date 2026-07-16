// pdb generator — a PodDisruptionBudget for a component. Applies only when the
// component declares a `pdb` block.
local helm = import '../helm.libsonnet';

{
  gvk: 'policy/v1/PodDisruptionBudget',
  when(c):: std.get(c, 'pdb', null) != null,
  // Only emit on clusters that actually expose policy/v1 (and the component is
  // enabled) — a whole-resource capability gate (#12).
  gate(c):: helm.gate.all([
    helm.gate.enabled(c),
    helm.gate.hasAPI('policy/v1/PodDisruptionBudget'),
  ]),
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
